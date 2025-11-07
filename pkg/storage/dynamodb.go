package storage

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	pkgerrors "github.com/TrevorEdris/retropie-utils/pkg/errors"
	"github.com/TrevorEdris/retropie-utils/pkg/fs"
	"github.com/TrevorEdris/retropie-utils/pkg/log"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/rotisserie/eris"
	"go.uber.org/zap"
)

type (
	DynamoDBClient struct {
		awsCfg             aws.Config
		client             *dynamodb.Client
		cfg                DynamoDBConfig
		resourcesValidated bool
		username           string
	}

	DynamoDBConfig struct {
		TableName              string
		Enabled                bool
		CreateMissingResources bool
	}

	FileMetadata struct {
		FileIdentifier   string `dynamodbav:"file-identifier"`
		S3Location       string `dynamodbav:"s3location"`
		OriginalFileName string `dynamodbav:"originalFileName"`
		FileDir          string `dynamodbav:"fileDir"`
		Username         string `dynamodbav:"username"`
		FileType         string `dynamodbav:"fileType"`
		LastModifiedTime int64  `dynamodbav:"lastModifiedTime"`
		CreatedAt        int64  `dynamodbav:"createdAt"`
	}
)

func NewDynamoDBClient(ctx context.Context, cfg DynamoDBConfig, username string) (*DynamoDBClient, error) {
	awscfg, err := newAwsConfig(ctx)
	if err != nil {
		return nil, err
	}
	client := dynamodb.NewFromConfig(awscfg)
	return &DynamoDBClient{
		awsCfg:   awscfg,
		client:   client,
		cfg:      cfg,
		username: username,
	}, nil
}

func (d *DynamoDBClient) Init(ctx context.Context) error {
	if !d.cfg.Enabled {
		return nil
	}

	// Validate required DynamoDB resources exist
	exist, err := d.checkIfTableExists(ctx)
	if err != nil {
		return err
	}

	// If they do not exist, create them if config is enabled
	if !exist && d.cfg.CreateMissingResources {
		err = d.createTable(ctx)
		if err != nil {
			return err
		}
		d.resourcesValidated = true
	}

	return nil
}

func (d *DynamoDBClient) StoreFileMetadata(ctx context.Context, s3Location string, file *fs.File, remoteDir string) error {
	if !d.cfg.Enabled {
		return nil
	}

	fileIdentifier := generateFileIdentifier(d.username, file.Dir, file.Name)
	fileTypeStr := fileTypeToString(file.FileType)
	now := time.Now().UnixMilli()

	metadata := FileMetadata{
		FileIdentifier:   fileIdentifier,
		S3Location:       s3Location,
		OriginalFileName: file.Name,
		FileDir:          file.Dir,
		Username:         d.username,
		FileType:         fileTypeStr,
		LastModifiedTime: now,
		CreatedAt:        now,
	}

	// Check if item exists to preserve createdAt
	existing, err := d.GetFileMetadata(ctx, fileIdentifier)
	if err == nil && existing != nil {
		metadata.CreatedAt = existing.CreatedAt
	}

	item, err := attributevalue.MarshalMap(metadata)
	if err != nil {
		return eris.Wrap(err, "failed to marshal file metadata")
	}

	_, err = d.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(d.cfg.TableName),
		Item:      item,
	})
	if err != nil {
		log.FromCtx(ctx).Error("Failed to store file metadata", zap.Error(err), zap.String("fileIdentifier", fileIdentifier))
		return eris.Wrap(err, "failed to store file metadata")
	}

	log.FromCtx(ctx).Info("Stored file metadata", zap.String("fileIdentifier", fileIdentifier), zap.String("s3Location", s3Location))
	return nil
}

func (d *DynamoDBClient) GetFileMetadata(ctx context.Context, fileIdentifier string) (*FileMetadata, error) {
	if !d.cfg.Enabled {
		return nil, pkgerrors.NotImplementedError
	}

	result, err := d.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(d.cfg.TableName),
		Key: map[string]types.AttributeValue{
			"file-identifier": &types.AttributeValueMemberS{
				Value: fileIdentifier,
			},
		},
	})
	if err != nil {
		return nil, eris.Wrap(err, "failed to get file metadata")
	}

	if result.Item == nil {
		return nil, nil
	}

	var metadata FileMetadata
	err = attributevalue.UnmarshalMap(result.Item, &metadata)
	if err != nil {
		return nil, eris.Wrap(err, "failed to unmarshal file metadata")
	}

	return &metadata, nil
}

func (d *DynamoDBClient) GetFileMetadataByFile(ctx context.Context, file *fs.File) (*FileMetadata, error) {
	fileIdentifier := generateFileIdentifier(d.username, file.Dir, file.Name)
	return d.GetFileMetadata(ctx, fileIdentifier)
}

func (d *DynamoDBClient) checkIfTableExists(ctx context.Context) (bool, error) {
	_, err := d.client.DescribeTable(ctx, &dynamodb.DescribeTableInput{
		TableName: aws.String(d.cfg.TableName),
	})
	if err == nil {
		log.FromCtx(ctx).Info("DynamoDB table exists", zap.String("table", d.cfg.TableName))
		return true, nil
	}

	var resourceNotFoundErr *types.ResourceNotFoundException
	if errors.As(err, &resourceNotFoundErr) {
		return false, nil
	}
	return false, err
}

func (d *DynamoDBClient) createTable(ctx context.Context) error {
	_, err := d.client.CreateTable(ctx, &dynamodb.CreateTableInput{
		TableName: aws.String(d.cfg.TableName),
		AttributeDefinitions: []types.AttributeDefinition{
			{
				AttributeName: aws.String("file-identifier"),
				AttributeType: types.ScalarAttributeTypeS,
			},
		},
		KeySchema: []types.KeySchemaElement{
			{
				AttributeName: aws.String("file-identifier"),
				KeyType:       types.KeyTypeHash,
			},
		},
		BillingMode: types.BillingModePayPerRequest,
	})
	if err != nil {
		log.FromCtx(ctx).Error("Failed to create DynamoDB table", zap.String("table", d.cfg.TableName), zap.Error(err))
		return err
	}

	// Wait for table to be active
	waiter := dynamodb.NewTableExistsWaiter(d.client)
	err = waiter.Wait(ctx, &dynamodb.DescribeTableInput{
		TableName: aws.String(d.cfg.TableName),
	}, 5*time.Minute)
	if err != nil {
		log.FromCtx(ctx).Error("Failed to wait for table to be active", zap.String("table", d.cfg.TableName), zap.Error(err))
		return err
	}

	log.FromCtx(ctx).Info("Successfully created DynamoDB table", zap.String("table", d.cfg.TableName))
	return nil
}

// generateFileIdentifier creates a unique identifier for a file
// Format: {username}#{fileDir}#{normalizedFileName}
func generateFileIdentifier(username, fileDir, fileName string) string {
	normalizedFileName := normalizeFileName(fileName)
	return fmt.Sprintf("%s#%s#%s", username, fileDir, normalizedFileName)
}

// normalizeFileName normalizes a filename for use in DynamoDB keys
// Converts spaces to underscores and converts to lowercase
func normalizeFileName(fileName string) string {
	normalized := strings.ReplaceAll(fileName, " ", "_")
	normalized = strings.ToLower(normalized)
	return normalized
}

// fileTypeToString converts FileType to string
func fileTypeToString(fileType fs.FileType) string {
	switch fileType {
	case fs.Rom:
		return "rom"
	case fs.Save:
		return "save"
	case fs.State:
		return "state"
	default:
		return "other"
	}
}


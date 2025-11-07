# retropie-utils
Various utilities I use to enhance the UX of my retropie

## Functionality

TODO

## Developing

### Requirements

**Note:** These docs assume you're using a Linux environment. A majority of development has been done via wsl2 running Ubuntu.

- [Docker](https://docs.docker.com/engine/install/)
- Make (`sudo apt-get install make`)
- [Go 1.23](https://github.com/moovweb/gvm?tab=readme-ov-file#installing)

### Running

**`make dev`**

Start the syncer alongside a local "AWS" environment.

```
❯ make dev
docker-compose -f docker-compose.dev.yaml up
Starting localstack ... done
Starting dynamodb-admin ... done
Starting syncer         ... done
Attaching to localstack, syncer, dynamodb-admin
localstack        |
localstack        | LocalStack version: 3.1.1.dev
syncer            |
syncer            |   __    _   ___
syncer            |  / /\  | | | |_)
syncer            | /_/--\ |_| |_| \_ v1.49.0, built with Go go1.23
syncer            |
syncer            | watching .
localstack        | LocalStack Docker container id: 1ea8d5445f5e
localstack        | LocalStack build date: 2024-02-05
localstack        | LocalStack build git hash: fee871a1
localstack        |
syncer            | watching _output
syncer            | watching bin
syncer            | watching pkg
. . .
localstack        | 2024-02-05T19:24:27.377  INFO --- [   asgi_gw_0] localstack.request.aws     : AWS s3.HeadBucket => 404 (NoSuchBucket)
localstack        | 2024-02-05T19:24:27.384  INFO --- [   asgi_gw_0] localstack.request.aws     : AWS s3.CreateBucket => 200
syncer            | 2024-02-05T19:24:27.385Z    INFO    syncer/syncer.go:64     Looking for roms in subfolders  {"directory": "/app/tools/syncer/example/roms"}
syncer            | 2024-02-05T19:24:27.385Z    INFO    storage/s3.go:101       Successfully created bucket     {"bucket": "retropie-backups"}
syncer            | 2024-02-05T19:24:27.385Z    INFO    syncer/syncer.go:73     Syncs enabled   {"roms": false, "saves": true, "states": true}
syncer            | 2024-02-05T19:24:27.385Z    INFO    syncer/syncer.go:82     Syncing saves
syncer            | 2024-02-05T19:24:27.385Z    INFO    syncer/syncer.go:107    Found 1 matching files
syncer            | 2024-02-05T19:24:27.385Z    INFO    storage/s3.go:121       Uploading /app/tools/syncer/example/roms/gba/Pokemon Fire Red.sav to retropie-backups/2024/02/05/19/gba/Pokemon Fire Red.sav
localstack        | 2024-02-05T19:24:27.394  INFO --- [   asgi_gw_0] localstack.request.aws     : AWS s3.PutObject => 200
syncer            | 2024-02-05T19:24:27.395Z    INFO    syncer/syncer.go:89     Syncing states
syncer            | 2024-02-05T19:24:27.395Z    INFO    syncer/syncer.go:107    Found 1 matching files
syncer            | 2024-02-05T19:24:27.395Z    INFO    storage/s3.go:121       Uploading /app/tools/syncer/example/roms/gb/Pokemon Blue.state to retropie-backups/2024/02/05/19/gb/Pokemon Blue.state
localstack        | 2024-02-05T19:24:27.398  INFO --- [   asgi_gw_0] localstack.request.aws     : AWS s3.PutObject => 200
syncer            | Process Exit with Code 0
```

### Testing

**`make install-tools`**

This will install the required tools for testing

**`make test`**

This will run the tests

```
❯ make test
ginkgo -v ./...
Running Suite: Errors Suite - /home/tedris/src/github.com/TrevorEdris/retropie-utils/pkg/errors
===============================================================================================
Random Seed: 1707161439

Will run 0 of 0 specs

Ran 0 of 0 Specs in 0.000 seconds
SUCCESS! -- 0 Passed | 0 Failed | 0 Pending | 0 Skipped
PASS
Running Suite: Fs Suite - /home/tedris/src/github.com/TrevorEdris/retropie-utils/pkg/fs
=======================================================================================
Random Seed: 1707161439

Will run 5 of 5 specs
------------------------------
File parses FileType correctly
/home/tedris/src/github.com/TrevorEdris/retropie-utils/pkg/fs/file_test.go:13
• [0.000 seconds]
------------------------------
Directory when the directory is flat lists all files in the directory
/home/tedris/src/github.com/TrevorEdris/retropie-utils/pkg/fs/directory_test.go:54
• [0.000 seconds]
------------------------------
Directory when the directory is flat gets matching files
/home/tedris/src/github.com/TrevorEdris/retropie-utils/pkg/fs/directory_test.go:63
• [0.000 seconds]
------------------------------
Directory when subdirectories exist lists all files in subdirectories also
/home/tedris/src/github.com/TrevorEdris/retropie-utils/pkg/fs/directory_test.go:116
• [0.001 seconds]
------------------------------
Directory when subdirectories exist gets matching files
/home/tedris/src/github.com/TrevorEdris/retropie-utils/pkg/fs/directory_test.go:125
• [0.001 seconds]
------------------------------

Ran 5 of 5 Specs in 0.002 seconds
SUCCESS! -- 5 Passed | 0 Failed | 0 Pending | 0 Skipped
PASS
. . .
Ginkgo ran 3 suites in 943.9503ms
Test Suite Passed
```

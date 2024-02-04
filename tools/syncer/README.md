# syncer

Sync files with a remote location.

## Installation

```
go install github.com/TrevorEdris/retropie-utils/tools/syncer@latest
```

## Usage

### Generate a configuration file

```
syncer config init
Created /home/tedris/.syncer/config.example.yaml
```

Copy the example config to `${HOME}/.syncer/config.ymal` and modify the values to fit your needs

### Sync files

```
syncer sync
Found existing config file: /home/tedris/.syncer/config.yaml
Running sync with config: storage:
    googledrive:
        enabled: false
    s3:
        enabled: true
        bucket: tedris-retropie-backups
    sftp:
        enabled: false
        username: ""
        password: ""
        port: 0
        remotedir: ""
romsfolder: /home/tedris/RetroPie/roms
sync:
    roms: true
    saves: true
    states: true
{"level":"info","ts":1702908442.6506999,"caller":"syncer/syncer.go:68","msg":"Looking for roms in subfolders","directory":"/home/tedris/RetroPie/roms"}
{"level":"info","ts":1702908442.6508422,"caller":"syncer/syncer.go:77","msg":"Syncs enabled","roms":true,"saves":true,"states":true}
{"level":"info","ts":1702908442.6508741,"caller":"syncer/syncer.go:79","msg":"Syncing ROMs"}
{"level":"info","ts":1702908442.650902,"caller":"syncer/syncer.go:111","msg":"Found 3 matching files"}
{"level":"info","ts":1702908442.6509194,"caller":"storage/s3.go:62","msg":"Uploading /home/tedris/RetroPie/roms/gb/Legend of Zelda, The - Link's Awakening (V1.2) (U) [!].gb to tedris-retropie-backups/2023/12/18/09/gb/Legend of Zelda, The - Link's Awakening (V1.2) (U) [!].gb"}
{"level":"info","ts":1702908443.3166378,"caller":"storage/s3.go:62","msg":"Uploading /home/tedris/RetroPie/roms/gb/Pokemon Blue (UA) [S][BF1].gb to tedris-retropie-backups/2023/12/18/09/gb/Pokemon Blue (UA) [S][BF1].gb"}
{"level":"info","ts":1702908443.6933835,"caller":"storage/s3.go:62","msg":"Uploading /home/tedris/RetroPie/roms/gb/Pokemon Red.gb to tedris-retropie-backups/2023/12/18/09/gb/Pokemon Red.gb"}
{"level":"info","ts":1702908444.117508,"caller":"syncer/syncer.go:86","msg":"Syncing saves"}
{"level":"info","ts":1702908444.1175764,"caller":"syncer/syncer.go:111","msg":"Found 3 matching files"}
{"level":"info","ts":1702908444.1176243,"caller":"storage/s3.go:62","msg":"Uploading /home/tedris/RetroPie/roms/gb/Legend of Zelda, The - Link's Awakening (V1.2) (U) [!].srm to tedris-retropie-backups/2023/12/18/09/gb/Legend of Zelda, The - Link's Awakening (V1.2) (U) [!].srm"}
{"level":"info","ts":1702908444.2486207,"caller":"storage/s3.go:62","msg":"Uploading /home/tedris/RetroPie/roms/gb/Pokemon Blue (UA) [S][BF1].srm to tedris-retropie-backups/2023/12/18/09/gb/Pokemon Blue (UA) [S][BF1].srm"}
{"level":"info","ts":1702908444.3632004,"caller":"storage/s3.go:62","msg":"Uploading /home/tedris/RetroPie/roms/gb/Pokemon Red.srm to tedris-retropie-backups/2023/12/18/09/gb/Pokemon Red.srm"}
{"level":"info","ts":1702908444.5188148,"caller":"syncer/syncer.go:93","msg":"Syncing states"}
{"level":"info","ts":1702908444.5188642,"caller":"syncer/syncer.go:111","msg":"Found 1 matching files"}
{"level":"info","ts":1702908444.5189154,"caller":"storage/s3.go:62","msg":"Uploading /home/tedris/RetroPie/roms/gb/Legend of Zelda, The - Link's Awakening (V1.2) (U) [!].state to tedris-retropie-backups/2023/12/18/09/gb/Legend of Zelda, The - Link's Awakening (V1.2) (U) [!].state"}
```

## TODO

- [X] Upload files to remote location
- [ ] Download the newest version of a file from the remote location
- [ ] Documentation

# Virtual Printer Processor
Well, my brother asked me to make a program that can print a file as an image to a file server using HTTP/S. 
I went a bit overboard and made a whole flexible configurable engine that can process the print job in any way you want. 
He wanted it for Windows, but I added support for Linux too(in a way).
Since I did it on a whim, I didn't write a lot of tests for it.

## Features
- Process print jobs in any way you want(within supported handlers)
- Retry mechanism for jobs processing
- CopyOnWrite for jobs processing to prevent data corruption
- Write ahead logging
- Flexible configuration
- Support for Windows and Linux

## Configuration
The configuration is done using a yaml file.
Values support the [expr](https://github.com/expr-lang/expr) language for more flexibility. 
They support it in a way that you have to use `${}` to call it.
For example: `${$env["SomeVar"] == 1}`. 
I expanded it a bit more so you have 2 more functions:
* `getEnv` - to get an environment variable. For example: `${getEnv("SomeVar")}`
* `uuid` - to generate a UUID. For example: `${uuid()}`

Whatever is in `config.yaml` in this repo is the configuration I conjured up for my brother's request.

Each handler gets a metadata map that it can write to and read from. You can also utilize it using the `${$env["MyVar]}` expression.
For example, if `WriteFile` writes to `WriteFile.OutputPath`, you can use it in the configuration like this: `${$env["WriteFile.OutputPath"]}`.
It can be escaped using `\$` if you want to use it as a string. For example: `"\${$env["WriteFile.OutputPath"]}"` which will output: `${$env["WriteFile.OutputPath"]}`.

### Example
```yaml
workdir: '${getEnv("TMP") != "" ? getEnv("TMP") : nil ?? "/tmp"}/MyVirtualPrinter'
write_ahead_logging:
  enabled: true
  max_size_mb: 2
  max_backups: 3
  max_age_days: 30
logs:
  level: "info"
  filename: "logs/app.log"
  max_size_mb: 2
  max_backups: 3
  max_age_days: 30
printer:
  name: MyPrinter
  monitor_interval_ms: 100 # interval to check for new print jobs to avoid high CPU usage
engine:
  max_workers: 2 # max number of workers to process the print jobs
  ignore_recovery_errors: false # if true, will ignore errors when trying to recover the engine state
  handlers: # list of handlers to process the print job
    - name: WriteFile # name of the handler
      config: # configuration for the handler
        output: '${getEnv("TMP") != "" ? getEnv("TMP") : nil ?? "/tmp"}/MyVirtualPrinter/tmpfile/${uuid()}.xps'
    - name: UploadHTTP
      config:
        url: https://api.whatsapp.com/send?phone=1234567890&text=Hello%20World
        method: POST
        type: multipart
        multipart_field_name: file
        multipart_filename: '${$env["WriteFile.OutputPath"]}'
      retry:
        max_retries: 3
        backoff_interval: 1 # back off interval in seconds
```

## Handlers
### WriteFile
Writes the object's contents to a file.
#### Configuration
- `output` - the output file path. Supports expressions.

#### Metadata:
Writes:
- `WriteFile.OutputPath` - the output file path.

### RunExecutable
Runs an executable.

#### Configuration
- `executable` - the executable path. Supports expressions.
- `args` - the arguments for the executable. Supports expressions.

### ReadFile
Reads a file's contents and writes it to the object.
#### Configuration
- `input` - the input file path. Supports expressions.
- `remove_source` - whether to remove the source file after reading it. Doesn't support expressions.

### MergePNGs
Merges multiple PNG files into one. It was developed with MuPDF in mind.
When MuPDF converts a file to PNG, it creates multiple PNG files for each page. This handler merges them into one, 
while assuming that each file is just the `{originalFileName}{pageNumber}.png`.

#### Configuration
- `input_file` - the input file path. It will be assumed that if the name is `file.png`, the pages are named `file1.png`, `file2.png`, etc. Supports expressions.
- `output_file` - the output file path. Supports expressions.
- `remove_old_files` - whether to remove the old files after merging them. Doesn't support expressions.

#### Metadata:
Writes:
- `MergePNGs.OutputFile` - the output file path.

### UploadHTTP
Uploads the object to an HTTP server.
#### Configuration
- `url` - the URL to upload the object to. Supports expressions.
- `method` - the HTTP method to use. Supports expressions.
- `type` - the type of the upload. Can be `multipart` or `base64`. Doesn't support expressions.
- `put_response_as_contents` - whether to put the response as the object's contents. Doesn't support expressions.
- `multipart_field_name` - the field name for the multipart upload. Supports expressions.
- `multipart_filename` - the filename for the multipart upload. If empty, will use multipart_field_name. Supports expressions.
- `headers` - the headers to send. Supports expressions.
- `base64_body_format` - the format of the body when using base64. To enter the base64 string, use `{{.Base64Contents}}`. For example: `{"data": "{{.Base64Contents}}"}` Supports expressions.
- `write_response_to_metadata` - whether to write the response body to the metadata. Doesn't support expressions.

#### Metadata:
Writes:
- `UploadHTTP.ResponseStatusCode` - the response status code.
- `UploadHTTP.ResponseBody` - the response body(if `write_response_to_metadata` is true).
- `UploadHTTP.ResponseHeaders` - the response headers(if `write_response_to_metadata` is true).
- `UploadHTTP.URL` - the URL used for the upload.

## Build
### Windows
```cmd
set CGO_ENABLED=1
set GO111MODULE=on
go build -ldflags "-H=windowsgui" -o app.exe
```

### Linux
```bash
sudo apt-get install libcups2-dev libayatana-appindicator3-dev printer-driver-cups-pdf
CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -o app
```


## Deep Dive
### The print processor
The print processor is a simple program that listens for print jobs and processes them.
Of course, that is true in case of windows. In Linux, it's a bit different.
In Windows, the program listens for print jobs using the `win32` API.
In Linux, it uses `cups-pdf` and listens for new files in the `~/PDF` folder.
Then the processor passes it to the engine using a channel to process it.

### The Engine
The engine is the core of the program. 
* It processes the print job using the handlers.
* It generates an ID for each handler when parsing them from the configuration.
* It generates the ID using the previous handler's ID and the handler's name.
In that way, when passing the id to the WAL, it can recover the state of the engine.
* Once the engine starts running, it will try to recover the state from the WAL.
If the configuration changes in a way to disrupts the previous order, it will not be able to proceed(unless the `ignore_recovery_errors` is set to true).
* For each job that is coming to the engine, it will copy the job's file to the workdir contents folder. 
Then it will pass the job to the workers to process it.
* For each handler, it will first log in the WAL that it started processing the job.
* Then, it will deep copy the flow metadata and pass it to the handler.
* If the handler fails, it will log the error in the WAL and retry the job(if the handler has a retry mechanism).
* If the handler succeeds, it will log the success in the WAL and pass the job(along with the new metadata) to the next handler.
* If the job passes all the handlers, it will be considered successful and the engine will delete the job's file from the workdir contents folder.
* Each handler gets a FileHandler that it can use to read and write the job's file. Although, it doesn't exactly read and write the direct file, but a copy of it.
* The engine uses a CopyOnWrite mechanism to prevent data corruption.
* If the handler opened a `Write()` stream, it will copy the file to a new file and pass the new file to the handler.
* If not, the same file will be used for the next handler.

Pseudo-code for the fileHandler:
```
    write() {
        write = true
        open write stream to file
        return the stream
    }
    
    read() {
        open read stream to file
        return the stream
        
    }
    
    getNewHandler() {
        if write {
            remove old input file
            input = output file
        }
        
        return new FileHandler(input, generateNewOutputPathInContents())
    }
```

Pseudo-code for engine jobs processing(for synchronous processing):
```
   for each job in jobs {
        LogToWal("copying job to contents dir")
        err = CopyJobToContentsDir(job)
        if err != nil {
            log.error(err)
            continue
        }
        
        fileHandler = NewFileHandler(job)
        for each handler in handlers {
            LogToWal("processing job with handler")
            metadata = deepCopy(job.metadata)
            for retry in handler.retries {
                err = handler.Process(fileHandler, metadata)
                if err != nil {
                    log.error(err)
                    if retry {
                        continue
                    }
                    break
                }
            }
        }
        LogToWal("deleting job from contents dir")
        delete job from contents dir
    }
```


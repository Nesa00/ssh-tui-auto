

### ssh-auto/main
go mod init ssh-auto/main

### ssh-auto/util
go mod init ssh-auto/utils

### ssh-auto/TUI
go mod init ssh-auto/tui


### ssh-auto/main
go mod edit -replace ssh-auto/utils=../utils
go mod edit -replace ssh-auto/tui=../tui

go get ssh-auto/utils
go get ssh-auto/tui

go mod tidy


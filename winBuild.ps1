go build -o builds/drop.exe .
go build -ldflags="-H=windowsgui" -o builds/dropw.exe .
Write-Host "Built drop.exe and dropw.exe"
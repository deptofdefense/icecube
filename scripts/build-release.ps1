$dir = Split-Path -parent $PSCommandpath

$os = "darwin freebsd linux openbsd solaris windows"
if ($Args.Count -gt 0) {
    $os = $Args[0]
}
$arch = "386 amd64 arm arm64"
if ($Args.Count -gt 1) {
    $arch = $Args[1]
}

$osarch = "!darwin/arm !darwin/386 !freebsd/arm64 !openbsd/arm !openbsd/arm64 !solaris/386 !solaris/arm !solaris/arm64"

$env:CGO_ENABLED = "0"

$env:GOFLAGS = "-mod=readonly"

Invoke-Expression "cmd.exe /c go mod download"

Invoke-Expression "cmd.exe /c $dir\..\bin\gox.exe -os=""$os"" -arch=""$arch"" -osarch=""$osarch"" -ldflags ""-s -w"" -output ""bin/{{.Dir}}_{{.OS}}_{{.Arch}}"" github.com/deptofdefense/icecube/cmd/icecube"
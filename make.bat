@echo off

rem # =================================================================
rem #
rem # Work of the U.S. Department of Defense, Defense Digital Service.
rem # Released as open source under the MIT License.  See LICENSE file.
rem #
rem # =================================================================

rem isolate changes to local environment
setlocal

rem update PATH to include local bin folder
PATH=%~dp0bin;%~dp0scripts;%PATH%


rem set common variables for targets

set "USAGE=Usage: %~n0 [bin\icecube.exe|clean|fmt|help|imports|staticcheck|tidy]"

rem if no target, then print usage and exit
if [%1]==[] (
  echo|set /p="%USAGE%"
  exit /B 1
)

if %1%==bin\gox.exe (

  rem create local bin folder if it doesn't exist
  if not exist "%~dp0bin" (
    mkdir %~dp0bin
  )

  go build -o bin/gox.exe github.com/mitchellh/gox

  exit /B 0
)

if %1%==bin\icecube.exe (

  rem create local bin folder if it doesn't exist
  if not exist "%~dp0bin" (
    mkdir %~dp0bin
  )

  go build -o bin/icecube.exe github.com/deptofdefense/icecube/cmd/icecube

  exit /B 0
)

if %1%==build_release (

  if not exist "%~dp0bin\gox.exe" (
      .\make.bat bin\gox.exe
  )

  powershell .\scripts\build-release.ps1

  exit /B 0
)

REM remove bin directory

if %1%==clean (

  if exist %~dp0bin (
    rd /s /q %~dp0bin
  )

  exit /B 0
)

if %1%==fmt (

  go fmt ./cmd/... ./pkg/...

  exit /B 0
)

if %1%==help (
  echo|set /p="%USAGE%"
  exit /B 1
)

if %1%==imports (

  rem create local bin folder if it doesn't exist
  if not exist "%~dp0bin" (
   mkdir %~dp0bin
  )

  if not exist "%~dp0bin\goimports.exe" (
    go build -o bin/goimports.exe golang.org/x/tools/cmd/goimports
  )

  .\bin\goimports.exe -w -local github.com/gruntwork-io/terratest,github.com/aws/aws-sdk-go,github.com/deptofdefense ./test ./pkg/tools

  exit /B 0
)

if %1%==serve_example (

    rem create local bin folder if it doesn't exist
    if not exist "%~dp0bin" (
      mkdir %~dp0bin
    )

    if not exist "%~dp0bin\icecube.exe" (
        .\make.bat bin\icecube.exe
    )

    if not exist "%~dp0temp" (
        mkdir %~dp0temp
    )

    if not exist "%~dp0temp\server.crt" (
        .\make.bat temp/server.crt
    )
    

    .\bin\icecube.exe serve ^
    --addr :8080 ^
    --server-cert temp\server.crt ^
    --server-key temp\server.key ^
    --root examples\public ^
    --unsafe --keylog temp\keylog

  exit /B 0
)

if %1%==serve_example_sni (

    rem create local bin folder if it doesn't exist
    if not exist "%~dp0bin" (
      mkdir %~dp0bin
    )

    if not exist "%~dp0bin\icecube.exe" (
        .\make.bat bin\icecube.exe
    )

    if not exist "%~dp0temp" (
        mkdir %~dp0temp
    )

    if not exist "%~dp0temp\server.crt" (
        .\make.bat temp/server.crt
    )

    if not exist "%~dp0temp\server_a.crt" (
        .\make.bat temp/server_a.crt
    )

    if not exist "%~dp0temp\server_b.crt" (
        .\make.bat temp/server_b.crt
    )
    

    .\bin\icecube.exe serve ^
    --addr :8080 ^
    --server-key-pairs "[[\"temp\\server_a.crt\", \"temp\\server_a.key\"], [\"temp\\server_b.crt\", \"temp\\server_b.key\"]]" ^
    --file-systems "[\"examples\\public\\a\", \"examples\\public\\b\"]" ^
    --sites "{\"a.localhost\": \"examples\\public\\a\",\"b.localhost\": \"examples\\public\\b\"}" ^
    --unsafe --keylog temp\keylog

  exit /B 0
)

if %1%==staticcheck (

  rem create local bin folder if it doesn't exist
  if not exist "%~dp0bin" (
    mkdir %~dp0bin
  )

  if not exist "%~dp0bin\staticcheck.exe" (
    go build -o bin/staticcheck.exe honnef.co/go/tools/cmd/staticcheck
  )

  .\bin\staticcheck.exe -checks all ./test

  exit /B 0
)

if %1%==temp/ca.crt (

    if not exist "%~dp0temp" (
        mkdir %~dp0temp
    )
    
    C:\Users\pdufour\AppData\Local\Programs\Git\mingw64\bin\openssl.exe req ^
    -batch ^
    -x509 ^
    -nodes ^
    -days 365 ^
    -newkey rsa:2048 ^
    -subj "/C=US/O=Atlantis/OU=Atlantis Digital Service/CN=icecubeca" ^
    -keyout temp/ca.key ^
    -out temp/ca.crt

    exit /B 0
)

if %1%==temp/ca.srl (

    if not exist "%~dp0temp" (
        mkdir %~dp0temp
    )
    
    echo 01 >temp/ca.srl

    exit /B 0
)

if %1%==temp/index.txt (

    if not exist "%~dp0temp" (
        mkdir %~dp0temp
    )
    
    copy NUL temp\index.txt

    exit /B 0
)

if %1%==temp/index.txt.attr (

    if not exist "%~dp0temp" (
        mkdir %~dp0temp
    )
    
	echo unique_subject = yes >temp/index.txt.attr

    exit /B 0
)


if %1%==temp/server.crt (

  if not exist "%~dp0temp" (
      mkdir %~dp0temp
  )

  if not exist "%~dp0temp\ca.crt" (
    .\make.bat temp/ca.crt
  )

  if not exist "%~dp0temp\ca.srl" (
    .\make.bat temp/ca.srl
  )

  if not exist "%~dp0temp\index.txt" (
    .\make.bat temp/index.txt
  )

  if not exist "%~dp0temp\index.txt.attr" (
      .\make.bat temp/index.txt.attr
  )

  C:\Users\pdufour\AppData\Local\Programs\Git\mingw64\bin\openssl.exe genrsa ^
  -out temp/server.key 2048

  C:\Users\pdufour\AppData\Local\Programs\Git\mingw64\bin\openssl.exe req ^
  -new ^
  -config examples/conf/openssl.cnf ^
  -key temp/server.key ^
  -subj "/C=US/O=Atlantis/OU=Atlantis Digital Service/CN=icecubelocal" ^
  -out temp/server.csr

  C:\Users\pdufour\AppData\Local\Programs\Git\mingw64\bin\openssl.exe ca ^
  -batch ^
  -config examples/conf/openssl.cnf ^
  -extensions server_ext ^
  -notext ^
  -in temp/server.csr ^
  -out temp/server.crt

  exit /B 0
)

if %1%==temp/server_a.crt (

  if not exist "%~dp0temp" (
      mkdir %~dp0temp
  )

  if not exist "%~dp0temp\ca.crt" (
    .\make.bat temp/ca.crt
  )

  if not exist "%~dp0temp\ca.srl" (
    .\make.bat temp/ca.srl
  )

  if not exist "%~dp0temp\index.txt" (
    .\make.bat temp/index.txt
  )

  if not exist "%~dp0temp\index.txt.attr" (
      .\make.bat temp/index.txt.attr
  )

  C:\Users\pdufour\AppData\Local\Programs\Git\mingw64\bin\openssl.exe genrsa ^
  -out temp/server_a.key 2048

  C:\Users\pdufour\AppData\Local\Programs\Git\mingw64\bin\openssl.exe req ^
  -new ^
  -config examples/conf/openssl.cnf ^
  -key temp/server_a.key ^
  -subj "/C=US/O=Atlantis/OU=Atlantis Digital Service/CN=a.localhost" ^
  -out temp/server_a.csr

  C:\Users\pdufour\AppData\Local\Programs\Git\mingw64\bin\openssl.exe ca ^
  -batch ^
  -config examples/conf/openssl.cnf ^
  -extensions server_ext ^
  -notext ^
  -in temp/server_a.csr ^
  -out temp/server_a.crt

  exit /B 0
)

if %1%==temp/server_b.crt (

  if not exist "%~dp0temp" (
      mkdir %~dp0temp
  )

  if not exist "%~dp0temp\ca.crt" (
    .\make.bat temp/ca.crt
  )

  if not exist "%~dp0temp\ca.srl" (
    .\make.bat temp/ca.srl
  )

  if not exist "%~dp0temp\index.txt" (
    .\make.bat temp/index.txt
  )

  if not exist "%~dp0temp\index.txt.attr" (
      .\make.bat temp/index.txt.attr
  )

  C:\Users\pdufour\AppData\Local\Programs\Git\mingw64\bin\openssl.exe genrsa ^
  -out temp/server_b.key 2048

  C:\Users\pdufour\AppData\Local\Programs\Git\mingw64\bin\openssl.exe req ^
  -new ^
  -config examples/conf/openssl.cnf ^
  -key temp/server_b.key ^
  -subj "/C=US/O=Atlantis/OU=Atlantis Digital Service/CN=b.localhost" ^
  -out temp/server_b.csr

  C:\Users\pdufour\AppData\Local\Programs\Git\mingw64\bin\openssl.exe ca ^
  -batch ^
  -config examples/conf/openssl.cnf ^
  -extensions server_ext ^
  -notext ^
  -in temp/server_b.csr ^
  -out temp/server_b.crt

  exit /B 0
)

if %1%==tidy (

  go mod tidy

  exit /B 0
)

if %1%==vet (

  go vet github.com/deptofdefense/icecube/pkg/...
  go vet github.com/deptofdefense/icecube/cmd/...

  exit /B 0
)

echo|set /p="%USAGE%"
exit /B 1
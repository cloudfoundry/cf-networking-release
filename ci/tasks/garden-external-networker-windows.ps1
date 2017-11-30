$ErrorActionPreference = "Stop";
trap { $host.SetShouldExit(1) }

$env:GOPATH = "$PWD/cf-networking-release"
$env:PATH = $env:GOPATH + "/bin;C:/go/bin;" + $env:PATH

if ((Get-Command "go.exe" -ErrorAction SilentlyContinue) -eq $null) {
  Write-Host "Installing Go"
  Invoke-WebRequest https://storage.googleapis.com/golang/go1.8.3.windows-amd64.msi -OutFile go.msi

  $p = Start-Process -FilePath "msiexec" -ArgumentList "/passive /norestart /i go.msi" -Wait -PassThru

  if($p.ExitCode -ne 0) {
    throw "Golang MSI installation process returned error code: $($p.ExitCode)"
  }

  Write-Host "Installed Go"
}
go.exe version

cd cf-networking-release

go.exe install ./src/github.com/onsi/ginkgo/ginkgo
if ($LastExitCode -ne 0) {
    throw "Ginkgo installation process returned error code: $LastExitCode"
}

cd src/garden-external-networker

ginkgo.exe -r -keepGoing

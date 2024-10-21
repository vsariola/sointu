  if ($args -contains "native") {

    Write-Host "Build VST with ASM synth"
    go build -buildmode=c-shared -tags="plugin","native" -o sointu-vsti.dll .\cmd\sointu-vsti\

  } elseif ($args -contains "go") {

    Write-Host "Build VST with GO synth"
    go build -buildmode=c-shared -tags="plugin" -o sointu-vsti.dll .\cmd\sointu-vsti\

  } else {
    Write-Host "specify either ""native"" or ""go"" argument."
  }


cd build
ninja sointu
cd ..

if ($LASTEXITCODE -eq 0) {
  if ($args -contains "native") {

    Write-Host "Render with ASM synth"
    go run -tags=native .\cmd\sointu-play\main.go -w .\examples\envelopexp_dev.yml

  } elseif ($args -contains "go") {

    Write-Host "Render with GO synth"
    go run .\cmd\sointu-play\main.go -w .\examples\envelopexp_dev.yml

  } else {
    Write-Host "specify either ""native"" or ""go"" argument."
  }
}

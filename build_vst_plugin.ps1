# Specify "native" or "go" if you only want one VST version.

if ($args -notcontains "go") {

    Write-Host "Build VST with ASM synth"
    go build -buildmode=c-shared -tags="plugin","native" -o sointu-vsti-native.dll .\cmd\sointu-vsti\

}
if ($args -notcontains "native") {

    Write-Host "Build VST with GO synth"
    go build -buildmode=c-shared -tags="plugin" -o sointu-vsti-go.dll .\cmd\sointu-vsti\

}


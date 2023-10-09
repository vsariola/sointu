from distutils.command.build_ext import build_ext
from distutils.errors import (
    CCompilerError,
    DistutilsExecError,
    DistutilsPlatformError,
)
from distutils.core import Extension
from os.path import (
    dirname,
    join,
    abspath,
    exists,
    basename,
    splitext,
)
from os import mkdir
from subprocess import run
from platform import system
from sys import exit

class BuildFailed(Exception):
    pass

class ExtBuilder(build_ext):

    def run(self):
        try:
            build_ext.run(self)
        except (DistutilsPlatformError, FileNotFoundError):
            raise BuildFailed('File not found. Could not compile C extension.')

    def build_extension(self, ext):
        try:
            build_ext.build_extension(self, ext)
        except (CCompilerError, DistutilsExecError, DistutilsPlatformError, ValueError):
            raise BuildFailed('Could not compile C extension.')

def build(setup_kwargs):
    # Make sure the build directory exists and setup the
    # relative paths correctly.
    cwd = abspath(".")
    print("Running from:", cwd)

    current_source_dir = abspath(dirname(__file__))
    project_source_dir = abspath(join(current_source_dir, "..", "..", ".."))
    current_binary_dir = join(current_source_dir, 'build')
    if not exists(current_binary_dir):
        mkdir(current_binary_dir)
    host_is_windows = system() == "Windows"
    executable_suffix = ".exe" if host_is_windows else ""
    object_suffix = ".obj" if host_is_windows else ".o"

    # Build the sointu compiler first.
    compiler_executable = join(current_binary_dir, "sointu-compile{}".format(executable_suffix))
    result = run(
        args=[
            "go", "build",
            "-o", compiler_executable,
            "cmd/sointu-compile/main.go",
        ],
        cwd=project_source_dir,
        shell=True if host_is_windows else False,
    )
    if result.returncode != 0:
        print("sointu-compile build process exited with:", result.returncode)
        print(result.stdout)
        exit(1)

    track_file_name = abspath(join(current_source_dir, "../../patches/physics_girl_st.yml"))
    (track_name_base, _) = splitext(basename(track_file_name)) 
    print("Compiling track:", track_file_name)

    # Compile the track.
    sointu_compiler_arch = "amd64"
    track_asm_file = join(current_binary_dir, '{}.asm'.format(track_name_base))
    result = run(
        args=[
            compiler_executable,
            "-o", track_asm_file,
            "-arch={}".format(sointu_compiler_arch),
            track_file_name,
        ],
    )
    if result.returncode != 0:
        print("sointu-compile process exited with:", result.returncode)
        print(result.stdout)
        exit(1)

    # Assemble the track.
    nasm_abi = "Win64" if host_is_windows else "Elf64"
    track_object_file = join(current_binary_dir, '{}{}'.format(track_name_base, object_suffix))
    print("Assembling track asm source:", track_asm_file)
    result = run(
        args=[
            'nasm',
            '-o', track_object_file,
            '-f', nasm_abi,
            track_asm_file,
        ],
    )
    if result.returncode != 0:
        print("nasm process exited with:", result.returncode)
        print(result.stdout)
        exit(1)
    
    # Export the plugin.
    print("Linking object file into Python extension module:", track_object_file)
    setup_kwargs.update({
        "ext_modules": [
            Extension(
                "sointu",
                include_dirs=[
                    current_binary_dir,
                    current_source_dir,
                ],
                sources=[
                    "sointu.c",
                ],
                extra_compile_args=[
                    "-DTRACK_HEADER=\"{}.h\"".format(track_name_base),
                ] + ([
                    "-DWIN32",
                ] if host_is_windows else [
                    "-DUNIX",
                    "-fPIC",
                ]),
                extra_objects=[
                    track_object_file,
                ],
                extra_link_args=[
                    "dsound.lib",
                    "ws2_32.lib",
                    "ucrt.lib",
                    "user32.lib",
                ] if host_is_windows else [
                    "-z", "noexecstack",
                    "--no-pie",
                    "-lasound",
                    "-lpthread",
                    "-lpython3.11",
                ],
            ),
        ],
        "cmdclass": {
            "build_ext": ExtBuilder,
        },
    })

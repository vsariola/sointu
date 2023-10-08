# -*- mode: python ; coding: utf-8 -*-
from os.path import abspath, join
from zipfile import ZipFile
from platform import system

moduleName = 'sointu_python'
rootPath = abspath('.')
buildPath = join(rootPath, 'build')
distPath = join(rootPath, 'dist')
sourcePath = join(rootPath, moduleName)

block_cipher = None

a = Analysis(
    [
        join(sourcePath, '__main__.py'),
    ],
    pathex=[],
    binaries=[],
    datas=[],
    hiddenimports=[],
    hookspath=[],
    hooksconfig={},
    runtime_hooks=[],
    excludes=[],
    win_no_prefer_redirects=False,
    win_private_assemblies=False,
    cipher=block_cipher,
    noarchive=False,
)
pyz = PYZ(a.pure, a.zipped_data, cipher=block_cipher)

exe = EXE(
    pyz,
    a.scripts,
    a.binaries,
    a.zipfiles,
    a.datas,
    [],
    name='{}'.format(moduleName),
    debug=False,
    bootloader_ignore_signals=False,
    strip=False,
    upx=True,
    upx_exclude=[],
    runtime_tmpdir=None,
    console=True,
    disable_windowed_traceback=False,
    argv_emulation=False,
    target_arch=None,
    codesign_identity=None,
    entitlements_file=None,
    icon=None,
)

exeFileName = '{}{}'.format(moduleName, '.exe' if system() == 'Windows' else '')
zipFileName = '{}-{}.zip'.format(moduleName, 'windows' if system() == 'Windows' else 'linux')
ZipFile(join(distPath, zipFileName), mode='w').write(join(distPath, exeFileName), arcname=exeFileName)

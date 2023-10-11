# Embed Sointu in Python
This is an example for embedding Sointu into Python code.

# Configure the track
Edit the `track` variable in `build.py` according to your needs.

# Build
* Install Python 3.11 and poetry.
* Download nasm and golang; place both of them in your system `PATH`.
* Enable cgo by downloading a gcc and placing it into your system `PATH`.
* Get the dependencies with `poetry install`.
* Run the player using `poetry run python -m sointu_python`.
* Pack everything into an executable using `poetry run pyinstaller sointu_python/sointu_python.spec`. The executable will be built in the `dist` subfolder.

# Rebuild after changes
* Rebuild the example track bindings with `poetry build`.
* Update the bindings module with `poetry install`.
* Proceed iteration.

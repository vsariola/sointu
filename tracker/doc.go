/*
Package tracker contains the data model for the Sointu tracker GUI.

The tracker package defines the Model struct, which holds the entire application
state, including the song data, instruments, effects, and large part of the UI
state.

The GUI does not modify the Model data directly, rather, there are types Action,
Bool, Int, String, List and Table which can be used to manipulate the model data
in a controlled way. For example, model.ShowLicense() returns an Action to show
the license to the user, which can be executed with model.ShowLicense().Do().

The various Actions and other data manipulation methods are grouped based on
their functionalities. For example, model.Instrument() groups all the ways to
manipulate the instrument(s). Similarly, model.Play() groups all the ways to
start and stop playback.

The method naming aims at API fluency. For example, model.Play().FromBeginning()
returns an Action to start playing the song from the beginning. Similarly,
model.Instrument().Add() returns an Action to add a new instrument to the song
and model.Instrument().List() returns a List of all the instruments.
*/
package tracker

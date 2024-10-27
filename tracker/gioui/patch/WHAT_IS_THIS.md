2024/10/27 qm210

As suggested by
https://github.com/vsariola/sointu/issues/156
and further by
https://github.com/vsariola/sointu/issues/151

the current Gio (0.7.1) Buttons (Clickables) have intrinsic behaviour
to re-trigger when SPACE is pressed.

This is so not what anyone using a Tracker expects, which is why for now,
this original Button component is copied here with that behaviour changed
https://github.com/gioui/gio/blob/v0.7.1/widget/button.go
https://github.com/gioui/gio/blob/v0.7.1/widget/material/button.go
https://github.com/gioui/gio/blob/v0.7.1/internal/f32color/rgba.go

This is obviously dangerous, because it decouples this Button from future
Gio releases, and our solution is a shady hack for now, but, 

we need to give the spacebar its space.

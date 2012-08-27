/*
Package wsevents is a simple WebSocket event handling abstraction.

All events are handled by a struct that implements the EventHandler interface.
Using the type of the struct passed into Handler, it looks at all of the names
of its methods and considers those that are prefixed with "On" to be event
handlers. These event handlers are called when events are received from the
client. For every new WebSocket connection, a new instance of the struct is
created.

Note: A simple JavaScript component still needs to be written for the client-side.
*/
package wsevents


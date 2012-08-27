/*
Package wsevents is a simple WebSocket event handling abstraction.

All events are handled by a struct that implements the EventHandler interface.
Using the type of the struct you pass in to Handler, it looks at all of the names
of its methods and considers those that are prefixed with "On" to be event
handlers. These event handlers are called when events are received from the
client. For every new WebSocket connection, a new instance of your struct is
created.
*/
package wsevents


package zzk

import "github.com/control-center/serviced/coordinator/client"

// Listener is intended to maintain a persisted zookeeper connection to
// maximize uptime of watched zookeeper nodes.
type Listener interface {
	Listen(cancel <-chan struct{}, conn client.Connection)
	Shutdown()
}

// Spawner manages the spawning of individual goroutines for managing nodes
// under a particular zookeeper path.
type Spawner interface {

	// SetConn sets the zookeeper connection
	SetConn(conn client.Connection)

	// Path returns the parent path of the zookeeper node whose children are
	// the target of spawn
	Path() string

	// Pre performs a synchronous action to occur before spawn
	Pre()

	// Spawn is intended to manage individual nodes that exist from Path()
	Spawn(cancel <-chan struct{}, n string)

	// Post presents the complete list of nodes that are children of Path() for
	// further processing and synchronization.
	Post(p map[string]struct{})
}

// Manage maintains the state of the listener
func Manage(shutdown <-chan struct{}, root string, l Listener) {
	defer l.Shutdown()

	logger := plog.WithField("zkroot", root)

	for {
		select {
		case conn := <-connect(root):
			if conn != nil {
				logger.Info("Acquired a client connection to zookeeper")
				l.Listen(shutdown, conn)
			}
		case <-shutdown:
		}

		// shutdown takes precedence
		select {
		case <-shutdown:
			return
		default:
		}
	}
}

// Listen manages spawning threads to handle nodes created under the parent
// path.
func Listen(shutdown <-chan struct{}, conn client.Connection, s Spawner) {
	var (
		cancel = make(chan struct{})
		exited = make(chan string)
		active = make(map[string]struct{})
	)

	logger := plog.WithField("zkpath", s.Path())

	// set the connection
	s.SetConn(conn)
	defer func() {
		close(cancel)
		for len(active) > 0 {
			delete(active, <-exited)
		}
	}()

	done = make(chan struct{})
	defer func() { close(done) }()
	for {
		// wait for the path to be available
		ok, ev, err := conn.ExistsW(s.Path(), done)
		if err != nil {
			logger.WithError(err).Error("Could not watch path")
			return
		}

		// get the path's children
		ch := []string{}
		if ok {
			ch, ev, err = conn.ChildrenW(s.Path(), done)
			if err == client.ErrNoNode {
				// path was deleted, so we need to monitor the existance
				close(done)
				done = make(chan struct{})
				continue
			} else if err != nil {
				logger.WithError(err).Error("Could not watch path children")
				return
			}
		}

		// spawn a goroutine for each new node
		for _, n := range ch {
			if _, ok := active[n]; !ok {
				logger.WithField("node", n).Debug("Spawning a goroutine for node")
				s.Pre()
				active[n] = struct{}{}
				go func(n string) {
					s.Spawn(cancel, n)
					exited <- n
				}(n)
			}
		}

		// trigger post-processing actions (for orphaned nodes)
		s.Post(active)

		select {
		case <-ev:
		case n := <-exited:
			delete(active, n)
		case <-shutdown:
		}

		// shutdown takes precedence
		select {
		case <-shutdown:
			return
		default:
		}

		close(done)
		done = make(chan struct{})
	}
}

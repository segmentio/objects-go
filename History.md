
v0.0.1-webhookfn / 2019-08-30
=============================

  * Add client configurations
  * Move reader to within retry loop, means it will work after first request (#7)
  * Log actual request object when Set call fails.
  * Updating tableize method call.
  * Make close return an error, remove call to reset()
  * Use `atomic.CompareAndSwapInt64` in Close
  * Using unbuffered exit channel to fix race condition.
  * Close channel after sending signal to exit
  * Make NewConcurrentMap private
  * Add validation, Client Tests, Reset buffers on close
  * Update cmap.go
  * Change check if closed
  * Wait for completition
  * First commit
  * Initial commit

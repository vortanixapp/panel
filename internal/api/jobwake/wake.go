package jobwake

import "log"

var notify func(string)

func SetNotifier(fn func(string)) {
	notify = fn
}

func Notify(jobType string) {
	if notify == nil {
		return
	}
	notify(jobType)
}

func ConnectNATS(url string) func() {
	if url == "" {
		return func() {}
	}
	nc, err := natsConnect(url)
	if err != nil {
		log.Printf("jobwake: NATS disabled: %v", err)
		return func() {}
	}
	log.Printf("jobwake: NATS connected %s", url)
	SetNotifier(func(jobType string) {
		_ = nc.Publish("vortanix.jobs.wake", []byte(jobType))
	})
	return func() { nc.Close() }
}

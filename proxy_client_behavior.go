package reverseproxy

import "time"

type clientBehaviorType int

const (
	do clientBehaviorType = iota
	doDeadline
	doRedirects
	doTimeout
)

type clientBehavior struct {
	clientBehaviorType clientBehaviorType
	param              interface{}
}

func ClientDo() clientBehavior {
	return clientBehavior{
		clientBehaviorType: do,
	}
}

func ClientDoRedirects(param int) clientBehavior {
	return clientBehavior{
		clientBehaviorType: doRedirects,
		param:              param,
	}
}

func ClientDoDeadline(param time.Time) clientBehavior {
	return clientBehavior{
		clientBehaviorType: doDeadline,
		param:              param,
	}
}

func ClientDoTimeout(param time.Time) clientBehavior {
	return clientBehavior{
		clientBehaviorType: doTimeout,
		param:              param,
	}
}

package store

type SecurityGroup struct {
	Guid              string
	Name              string
	Rules             string
	StagingDefault    bool
	RunningDefault    bool
	StagingSpaceGuids []string
	RunningSpaceGuids []string
}

type Page struct {
	Limit int
	From  int
}

type Pagination struct {
	Next int
	Prev int
}

package main

import (
	"bytes"
	"fmt"
	"math/rand/v2"
	"sort"
)

// MAXSECS is the maximum number of seconds in a year.
const MAXSECS = 365 * 24 * 3600

const (
	// SIMU_SEED1 is used to make the random generator deterministic
	SIMU_SEED1 = 4321
	// SIMU_SEED2 is used to make the random generator deterministic
	SIMU_SEED2 = 384172
)

const (
	N_NODES              = 12
	N_ZONES              = 3
	N_ROLLING_UPGRADES   = 2
	MTBR_ROLLING_UPGRADE = 2 * 60
	IDLE_ROLLING_UPGRADE = 30
	N_ZONE_SHUTDOWNS     = 1
	MTBR_ZONE_SHUTDOWN   = 3 * 3600
	N_REBOOTS            = 3
	MTBR_REBOOT          = 5 * 60
	PROB_NET_FAILURE     = 25
	MTBR_NET_FAILURE     = 3600
	PROB_HW_FAILURE      = 6
	MTBR_HW_FAILURE      = 3 * 24 * 3600
	THROUGHPUT           = 50

	ZONE_SIZE = N_NODES / N_ZONES
	N_HISTO   = 11
)

// Interval represents an availability interval.
type Interval struct {
	beg, end uint32
	cnt      uint32
	ratio    float32
}

// Overlap checks the overlap between two intervals.
func (i Interval) Overlap(o Interval) bool {
	return i.end > o.beg && i.beg < o.end
}

// Contiguous checks the intervals are Contiguous.
func (i Interval) Contiguous(o Interval) bool {
	return i.end == o.beg || i.beg == o.end
}

// Include returns true if the interval include t.
func (i Interval) Include(t uint32) bool {
	return t >= i.beg && t <= i.end
}

// Normalize ensures the interval is within the range.
func (i *Interval) Normalize() {
	if i.end > MAXSECS {
		i.end = MAXSECS
	}
}

// Intervals is a slice of intervals.
type Intervals []Interval

// Reset cleans so the object can be reused.
func (s *Intervals) Reset() {
	*s = (*s)[:0]
}

// AddFailure adds a failure event avoiding collisions.
// Return false if an overlap check fails.
func (s *Intervals) AddFailure(t uint32, mttr uint32, check bool) bool {
	x := Interval{t, t + mttr, 0, 0.0}
	x.Normalize()
	if check && s.CheckCollision(x) {
		return false
	}
	*s = append(*s, x)
	return true
}

// CheckCollision returns true if an existing interval overlaps.
func (s Intervals) CheckCollision(x Interval) bool {
	for i := range s {
		if s[i].Overlap(x) {
			return true
		}
	}
	return false
}

// CheckCollisionTime returns true if t matches an existing interval.
func (s Intervals) CheckCollisionTime(t uint32) bool {
	for i := range s {
		if s[i].Include(t) {
			return true
		}
	}
	return false
}

// AddFailures adds multiple failures avoiding collisions.
func (s *Intervals) AddFailures(n int, r *rand.Rand, mttr uint32) {
	for n > 0 {
		if !s.AddFailure(r.Uint32N(MAXSECS), mttr, true) {
			continue
		}
		n--
	}
}

// FindNonFailureTime returns a timestamp which does not match an existing interval.
func (s Intervals) FindNonFailureTime(r *rand.Rand) uint32 {
	for {
		t := r.Uint32N(MAXSECS)
		if !s.CheckCollisionTime(t) {
			return t
		}
	}
}

// Normalize puts the list of intervals in canonical form
func (sp *Intervals) Normalize(sorted bool) {

	// Sort intervals
	s := *sp
	if len(s) == 0 {
		return
	}
	if !sorted {
		sort.Sort(s)
	}

	// Merge contiguous or overlapping intervals
	for i := 1; i < len(s); {
		if s[i].Overlap(s[i-1]) {
			if s[i-1].end < s[i].end {
				s[i-1].end = s[i].end
			}
			s = append(s[:i], s[i+1:]...)
			continue
		}
		if s[i].Contiguous(s[i-1]) {
			s[i-1].end = s[i].end
			s = append(s[:i], s[i+1:]...)
			continue
		}
		i++
	}
	*sp = s
}

// Equal returns true if the two objects are identical
func (s Intervals) Equal(other Intervals) bool {
	if len(s) != len(other) {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] != other[i] {
			return false
		}
	}
	return true
}

// MergeNodes merges two interval slices associated to two nodes of the same zone
func (s *Intervals) MergeNodes(a, b Intervals) {
	s.merge(a, b, func(x, y Interval) Interval {
		return Interval{x.beg, x.end, x.cnt, x.ratio + y.ratio}
	})
}

// MergeZones merges two interval slices associated to two zones of the same cluster
func (s *Intervals) MergeZones(a, b Intervals) {
	s.merge(a, b, func(x, y Interval) Interval {
		return Interval{x.beg, x.end, x.cnt + y.cnt, x.ratio * y.ratio}
	})
}

// merge applies the generic merge algorithm of two interval slices.
// The two slices must be sorted.
func (s *Intervals) merge(a, b Intervals, gen func(x, y Interval) Interval) {
	s.Reset()
	i, j := 0, 0
	var x Interval
	for i < len(a) && j < len(b) {
		ab, ae, bb, be := a[i].beg, a[i].end, b[j].beg, b[j].end
		switch {
		case ae <= bb:
			*s = append(*s, a[i])
			i++
		case be <= ab:
			*s = append(*s, b[j])
			j++
		case ab == bb:
			switch {
			case ae == be:
				*s = append(*s, gen(a[i], b[j]))
				i++
				j++
			case ae < be:
				*s = append(*s, gen(a[i], b[j]))
				b[j].beg = ae
				i++
			default:
				*s = append(*s, gen(b[j], a[i]))
				a[i].beg = be
				j++
			}
		case ab < bb:
			x = a[i]
			x.end, a[i].beg = bb, bb
			*s = append(*s, x)
		default:
			x = b[j]
			x.end, b[j].beg = ab, ab
			*s = append(*s, x)
		}
	}
	switch {
	case i < len(a):
		*s = append(*s, a[i:]...)
	case j < len(b):
		*s = append(*s, b[j:]...)
	}
}

// Implements sort interface.
func (s Intervals) Len() int           { return len(s) }
func (s Intervals) Less(i, j int) bool { return s[i].beg < s[j].beg }
func (s Intervals) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

// AvailabilityYear represents an availability for a given year.
type AvailabilityYear struct {
	nodes   []Intervals
	zones   []Intervals
	cluster Intervals
	tmp     Intervals
	res     Result
}

// NewAvailabilityYear creates a new object representing a full year of availability.
func NewAvailabilityYear() *AvailabilityYear {
	const NINIT = 8
	ret := &AvailabilityYear{
		nodes:   make([]Intervals, N_NODES),
		zones:   make([]Intervals, N_ZONES),
		cluster: make([]Interval, 0, NINIT),
		tmp:     make([]Interval, 0, NINIT),
	}
	for i := 0; i < N_NODES; i++ {
		ret.nodes[i] = make([]Interval, 0, NINIT)
	}
	for i := 0; i < N_ZONES; i++ {
		ret.zones[i] = make([]Interval, 0, NINIT)
	}
	return ret
}

// Reset re-initializes the object.
func (ay *AvailabilityYear) Reset() {
	for i := range ay.nodes {
		ay.nodes[i].Reset()
	}
	for i := range ay.zones {
		ay.zones[i].Reset()
	}
	ay.cluster.Reset()
	ay.tmp.Reset()
	ay.res.Reset()
}

// Build generates a simulation.
func (ay *AvailabilityYear) Build(r *rand.Rand) {
	ay.buildNodes(r)
	ay.buildGlobalEvents(r)
	ay.retrofitGlobalEvents()
}

// BuildNodes populates the initial node views.
func (ay *AvailabilityYear) buildNodes(r *rand.Rand) {

	// Each node suffers from N_REBOOTS unexpected reboot a year, resulting in MTBR_REBOOT secs outage per node.
	// Each node has a PROB_HW_FAILURE% chance a year to get a hardware failure resulting in MTBR_HW_FAILURE secs outage.
	for i := range ay.nodes {
		ay.nodes[i].AddFailures(N_REBOOTS, r, MTBR_REBOOT)
		if r.IntN(100) < PROB_HW_FAILURE {
			ay.nodes[i].AddFailures(1, r, MTBR_HW_FAILURE)
		}
	}

	// Each availability zone has a PROB_NET_FAILURE% chance a year to suffer from a network issue
	// making it unavailable for MTBR_NET_FAILURE secs.
	for iZ := 0; iZ < N_ZONES; iZ++ {
		if r.IntN(100) < PROB_NET_FAILURE {
			t := r.Uint32N(MAXSECS)
			for i := 0; i < ZONE_SIZE; i++ {
				ay.nodes[iZ*ZONE_SIZE+i].AddFailure(t, MTBR_NET_FAILURE, false)
			}
		}
	}

	// Build the global view
	for i := range ay.nodes {
		ay.cluster = append(ay.cluster, ay.nodes[i]...)
	}
}

// BuildGlobalEvents generate global events such as zone shutdown and rolling upgrades.
func (ay *AvailabilityYear) buildGlobalEvents(r *rand.Rand) {

	// Each availability zone is brought down N_ZONE_SHUTDOWNS times a year resulting in MTBR_ZONE_SHUTDOWN secs outage per zone.
	// The cluster is upgraded N_ROLLING_UPGRADES times a year using rolling upgrade, resulting in MTBR_ROLLING_UPGRADE secs outage
	// per node in sequence, separated by IDLE_ROLLING_UPGRADE secs idle periods.
	// Availability zone shutdowns and couchbase rolling upgrades are scheduled events, so:
	//   - they are mutually exclusive
	//   - they are not scheduled when there is already a node down for any reason
	for i := 0; i < N_ROLLING_UPGRADES; i++ {
		for {
			tRU := ay.cluster.FindNonFailureTime(r)
			dur := uint32((N_NODES-1)*(MTBR_ROLLING_UPGRADE+IDLE_ROLLING_UPGRADE) + MTBR_ROLLING_UPGRADE)
			if ay.tmp.AddFailure(tRU, dur, true) {
				break
			}
		}
	}
	for i := 0; i < N_ZONE_SHUTDOWNS*N_ZONES; i++ {
		for {
			tZS := ay.cluster.FindNonFailureTime(r)
			if ay.tmp.AddFailure(tZS, MTBR_ZONE_SHUTDOWN, true) {
				break
			}
		}
	}
}

// BuildNodes update the node views according to the global events.
func (ay *AvailabilityYear) retrofitGlobalEvents() {

	// Retrofit global events in the node views.
	idx := 0
	for i := 0; i < N_ROLLING_UPGRADES; i++ {
		tRU := ay.tmp[idx].beg
		for n := 0; n < N_NODES; n++ {
			ay.nodes[n].AddFailure(tRU, MTBR_ROLLING_UPGRADE, false)
			tRU += MTBR_ROLLING_UPGRADE + IDLE_ROLLING_UPGRADE
		}
		idx++
	}
	for iS := 0; iS < N_ZONE_SHUTDOWNS; iS++ {
		for iZ := 0; iZ < N_ZONES; iZ++ {
			tZS := ay.tmp[idx].beg
			for n := 0; n < ZONE_SIZE; n++ {
				ay.nodes[iZ*ZONE_SIZE+n].AddFailure(tZS, MTBR_ZONE_SHUTDOWN, false)
			}
			idx++
		}
	}
}

// Simulate calculate the result of the simulation.
func (ay *AvailabilityYear) Simulate() {
	ay.tmp.Reset()
	ay.cluster.Reset()
	ay.normalize()
	ay.simulateZones()
	ay.simulateCluster()
}

// BuildNodes update the node views according to the global events.
func (ay *AvailabilityYear) normalize() {
	for i := range ay.nodes {
		ay.nodes[i].Normalize(false)
		for j := range ay.nodes[i] {
			p := &(ay.nodes[i][j])
			p.cnt = 1
			p.ratio = 1.0 / float32(ZONE_SIZE)
		}
	}
}

// simulateZone calculates the result of the simulation for the zones.
func (ay *AvailabilityYear) simulateZones() {
	for iZ := range ay.zones {
		nodes := ay.nodes[iZ*ZONE_SIZE : (iZ+1)*ZONE_SIZE]
		z := ay.zones[iZ]
		z = append(z, nodes[0]...)
		for i := 1; i < len(nodes); i++ {
			ay.tmp.MergeNodes(z, nodes[i])
			z, ay.tmp = ay.tmp, z
		}
		ay.zones[iZ] = z
	}
}

// simulateCluster calculates the result of the simulation for the cluster.
func (ay *AvailabilityYear) simulateCluster() {
	c := ay.cluster
	c = append(c, ay.zones[0]...)
	for iZ := 1; iZ < len(ay.zones); iZ++ {
		ay.tmp.MergeZones(c, ay.zones[iZ])
		c, ay.tmp = ay.tmp, c
	}
	ay.cluster = c
}

// Evaluate analyzes the result of the simulation.
func (ay *AvailabilityYear) Evaluate() {
	ay.res.n++
	for i, x := range ay.cluster {
		if x.cnt > 0 {
			n := x.cnt - 1
			ay.res.outages[n].Update(x)
			if i == 0 || ay.cluster[i-1].end != x.beg || ay.cluster[i-1].cnt < x.cnt {
				ay.res.failures[n].Update(x)
			}
			if x.cnt > 1 || !almostEqual(x.ratio, 1.0/float32(ZONE_SIZE)) {
				ay.res.atLeast2.Update(x)
			}
		}
	}
}

func almostEqual(a, b float32) bool {
	// Note: this assume normalized numbers between 0 and 1
	diff := b - a
	if diff < 0 {
		diff = -diff
	}
	return diff < float32(1.0e-5)
}

// Statistic contains the metric values associated to a single event.
type Statistic struct {
	n, dur int
	rat    float64
}

// Aggregate sums statistics
func (s *Statistic) Aggregate(other *Statistic) {
	s.n += other.n
	s.dur += other.dur
	s.rat += other.rat
	if s.n < 0 || s.dur < 0 {
		panic("Statistic.Aggregate: integer overflow")
	}
}

// Update aggregates results with previous runs.
func (s *Statistic) Update(x Interval) {
	s.n++
	s.dur += int(x.end - x.beg)
	s.rat += float64(x.ratio)
}

// Result contains the result of a simulation run.
// It can also aggregate results of multiple runs.
type Result struct {
	n        int
	atLeast2 Statistic
	outages  [N_ZONES]Statistic
	failures [N_ZONES]Statistic
}

// Reset zeroes the result object.
func (r *Result) Reset() {
	*r = Result{}
}

// Aggregate sums statistics from other runs.
func (r *Result) Aggregate(other *Result) {
	r.n += other.n
	r.atLeast2.Aggregate(&(other.atLeast2))
	for i := range r.outages {
		r.outages[i].Aggregate(&(other.outages[i]))
	}
	for i := range r.failures {
		r.failures[i].Aggregate(&(other.failures[i]))
	}
}

// FinalResult contains aggregated results with probabilites.
type FinalResult struct {
	Result
	z1Cnt int
	z1Sum int
	proba [N_ZONES][N_HISTO]int
}

// Update aggregates statistics and maintain probability counters.
func (r *FinalResult) Update(other *Result) {

	r.Result.Aggregate(other)

	// Build an histogram for "at least 2" failures.
	// Expected zone shutdown are excluded.
	n := other.atLeast2.n - N_ZONE_SHUTDOWNS*N_ZONES
	r.calculate(0, n)

	// Node failure in a single zone, an histogram is useless.
	// Just calculate an average instead.
	n = other.failures[0].n
	if n > 0 {
		r.z1Cnt++
		r.z1Sum += n
	}

	// Failures in two or more zones, build an histogram.
	for i := 1; i < N_ZONES; i++ {
		n = other.failures[i].n
		r.calculate(i, n)
	}
}

func (r *FinalResult) calculate(i, n int) {
	if n > 0 {
		if n >= N_HISTO {
			n = 0
		}
		r.proba[i][n]++
	}
}

// Aggregate sums results from other runs
func (r *FinalResult) Aggregate(other *FinalResult) {

	r.Result.Aggregate(&(other.Result))

	r.z1Cnt += other.z1Cnt
	r.z1Sum += other.z1Sum

	for i := range r.proba {
		for j := range r.proba[i] {
			r.proba[i][j] += other.proba[i][j]
		}
	}
}

// Format generates a human-readable output
func (r *FinalResult) Format(f fmt.State, c rune) {

	fmt.Fprintf(f, "Number of simulations:            %d\n", r.Result.n)
	fmt.Fprintf(f, "Average node failures per year:   %.2f\n", float64(r.z1Sum)/float64(r.z1Cnt))
	fmt.Fprintln(f)

	r.displayRangeProb(f, N_ZONES)
	fmt.Fprintln(f)
	r.displayRangeProb(f, N_ZONES-1)
	fmt.Fprintln(f)
	r.displayRangeProb(f, 1)
}

// displayRangeProb displays probabilities
func (r *FinalResult) displayRangeProb(f fmt.State, nz int) {

	if nz == 1 {
		fmt.Fprintf(f, "Failures involving at least 2 nodes\n\n")
		displayAverages(f, nz, r.Result.atLeast2)
	} else {
		fmt.Fprintf(f, "Failures on %d zones\n\n", nz)
		displayAverages(f, nz, r.Result.outages[nz-1])
	}

	fmt.Fprintf(f, "\nHistogram of probability\n")

	t := r.proba[nz-1][:]
	count := float64(r.Result.n)
	more := true

	for i, x := range t[1:] {
		if x == 0 {
			more = false
			break
		}
		fmt.Fprintf(f, "At least %2d occurences:  %8.4f %%\n", i+1, 100.0*float64(sum(i+1, t))/count)
	}
	if more && t[0] != 0 {
		fmt.Fprintf(f, "More occurences:         %8.4f %%\n", 100.0*float64(t[0])/count)
	}
}

// displayAverages displays average metrics
func displayAverages(f fmt.State, nz int, s Statistic) {

	imp := s.rat / float64(s.n)
	dur := float64(s.dur) / float64(s.n)

	fmt.Fprintf(f, "Average %% of keys impacted:      %3.4f %%\n", 100.0*imp)
	fmt.Fprintf(f, "Average duration of the event:   %.0f secs\n", dur)

	if nz == N_ZONES {
		fmt.Fprintf(f, "Average number of records lost:  %3.4f\n", float64(THROUGHPUT)*imp)
	} else {
		fmt.Fprintf(f, "Average number of transactions:  %3.4f\n", float64(THROUGHPUT)*dur*imp)
	}
}

// sum calculates a sum of a slice
func sum(i int, t []int) int {
	n := t[0]
	for _, x := range t[i:] {
		n += x
	}
	return n
}

// Simulator is a Monte-Carlo simulation whose purpose is to calculate the probability of failures
// of a Couchbase cluster.
type Simulator struct {
	id  int
	pcg *rand.PCG
	r   *rand.Rand
	ay  *AvailabilityYear
	res FinalResult
}

// NewSimulator create a simulator instance
func NewSimulator(n int) *Simulator {

	pcg := rand.NewPCG(SIMU_SEED1, SIMU_SEED2)
	return &Simulator{
		id:  n,
		pcg: pcg,
		r:   rand.New(pcg),
		ay:  NewAvailabilityYear(),
	}

}

// Run starts and runs the simulation for n steps
func (s *Simulator) Run(n int) {
	s.pcg.Seed(SIMU_SEED1, SIMU_SEED2)
	for i := 0; i < n; i++ {
		s.ay.Reset()
		s.ay.Build(s.r)
		s.ay.Simulate()
		s.ay.Evaluate()
		s.res.Update(&(s.ay.res))
	}
}

// BenchSimulation consists in running steps of the Monte-Carlo simulation
type BenchSimulation struct {
	simu *Simulator
	buf  bytes.Buffer
}

// NewBenchSimulation created a benchmark instance
func NewBenchSimulation() *BenchSimulation {
	return &BenchSimulation{
		simu: NewSimulator(0),
	}
}

// Run executes n steps of the simulation and format results
func (b *BenchSimulation) Run() {
	b.simu.Run(100)
	fmt.Fprint(&b.buf, &b.simu.res)
	b.buf.Reset()
}

package tests

import (
	"bytes"
	"fmt"
	"math/rand"
	"time"

	"github.com/google/uuid"

	"github.com/getsentry/go-load-tester/utils"
)

type User struct {
	IpAddress string `json:"ip_address,omitempty"`
	Username  string `json:"username,omitempty"`
	Id        string `json:"id,omitempty"`
}

type Contexts struct {
	Os     OsContext     `json:"os,omitempty"`
	Device DeviceContext `json:"device,omitempty"`
	App    AppContext    `json:"app,omitempty"`
	Trace  TraceContext  `json:"trace,omitempty"`
}

type OsContext struct {
	Type          string `json:"type,omitempty"`
	Rooted        *bool  `json:"rooted,omitempty"`
	KernelVersion string `json:"kernel_version,omitempty"`
	Version       string `json:"version,omitempty"`
	Built         string `json:"built,omitempty"`
	Name          string `json:"name,omitempty"`
}

type DeviceContext struct {
	Name                string  `json:"name,omitempty"`
	Family              string  `json:"family,omitempty"`
	Model               string  `json:"model,omitempty"`
	ModelId             string  `json:"model_id,omitempty"`
	Arch                string  `json:"arch,omitempty"`
	BatteryLevel        float64 `json:"battery_level,omitempty"`
	Orientation         string  `json:"orientation,omitempty"`
	Manufacturer        string  `json:"manufacturer,omitempty"`
	Brand               string  `json:"brand,omitempty"`
	ScreenResolution    string  `json:"screen_resolution,omitempty"`
	ScreenDensity       uint64  `json:"screen_density,omitempty"`
	ScreenDpi           uint64  `json:"screen_dpi,omitempty"`
	Online              bool    `json:"online,omitempty"`
	Charging            bool    `json:"charging,omitempty"`
	LowMemory           bool    `json:"low_memory,omitempty"`
	Simulator           bool    `json:"simulator,omitempty"`
	MemorySize          uint64  `json:"memory_size,omitempty"`
	FreeMemory          uint64  `json:"free_memory,omitempty"`
	UsableMemory        uint64  `json:"usable_memory,omitempty"`
	StorageSize         uint64  `json:"storage_size,omitempty"`
	FreeStorage         uint64  `json:"free_storage,omitempty"`
	ExternalStorageSize uint64  `json:"external_storage_size,omitempty"`
	ExternalFreeStorage uint64  `json:"external_free_storage,omitempty"`
	BootTime            string  `json:"boot_time,omitempty"`
	Timezone            string  `json:"timezone,omitempty"`
	Type                string  `json:"type,omitempty"`
	// in locust tester but not in relay
	//ScreenWidthPixels  uint64
	//ScreenHeightPixels uint64
	//Locale             string
}

type AppContext struct {
	AppStartTime  string `json:"app_start_time,omitempty"`
	DeviceAppHash string `json:"device_app_hash,omitempty"`
	BuildType     string `json:"build_type,omitempty"`
	AppIdentifier string `json:"app_identifier,omitempty"`
	AppName       string `json:"app_name,omitempty"`
	AppVersion    string `json:"app_version,omitempty"`
	AppBuild      string `json:"app_build,omitempty"`
	Type          string `json:"type,omitempty"`
}

type TraceContext struct {
	Type          string  `json:"type,omitempty"`
	TraceId       string  `json:"trace_id,omitempty"`
	SpanId        string  `json:"span_id,omitempty"`
	ParentSpanId  string  `json:"parent_span_id,omitempty"`
	Op            string  `json:"op,omitempty"`
	Status        string  `json:"status,omitempty"`
	ExclusiveTime float64 `json:"exclusive_time,omitempty"`
}

type Breadcrumb struct {
	Timestamp float64 `json:"timestamp"`
	Ty        string  `json:"ty"`
	Category  string  `json:"category"`
	Level     string  `json:"level"`
	Message   string  `json:"message"`
	Data      any     `json:"data"`
	EventId   string  `json:"event_id"`
}

type Span struct {
	Timestamp      float64           `json:"timestamp,omitempty"`
	StartTimestamp float64           `json:"start_timestamp,omitempty"`
	ExclusiveTime  float64           `json:"exclusive_time,omitempty"`
	Description    string            `json:"description,omitempty"`
	Op             string            `json:"op,omitempty"`
	SpanId         string            `json:"span_id,omitempty"`
	ParentSpanId   string            `json:"parent_span_id,omitempty"`
	TraceId        string            `json:"trace_id,omitempty"`
	Status         string            `json:"status,omitempty"`
	Tags           map[string]string `json:"tags,omitempty"`
	Data           any               `json:"data,omitempty"`
}

func TransactionGenerator(job TransactionJob) func() Transaction {
	idGen := EventIdGenerator()
	relGen := ReleaseGenerator(job.NumReleases)
	transGen := func() string {
		if Flip() {
			return ""
		} else {
			return fmt.Sprintf("mytransaction%d", rand.Intn(100))
		}
	}
	userGen := UserGenerator(job.NumUsers)
	osGen := OsContextGenerator()
	deviceGen := DeviceContextGenerator()
	appGen := AppContextGenerator()
	traceGen := TraceContextGenerator(job.Operations)
	breadcrumbsGen := BreadcrumbsGenerator(job.MinBreadCrumbs, job.MaxBreadCrumbs, job.BreadcrumbCategories,
		job.BreadcrumbLevels, job.BreadcrumbsTypes, job.BreadcrumbMessages)
	measurementsGen := MeasurementsGenerator(job.Measurements)
	spansGen := SpansGenerator(job.MinSpans, job.MaxSpans, job.Operations)

	transactionDurationMin := job.TransactionDurationMin
	transactionDurationMax := job.TransactionDurationMax
	transactionTimestampSpread := job.TransactionTimestampSpread
	transactionRange := transactionDurationMax - transactionDurationMin

	return func() Transaction {
		trace := traceGen()
		transactionId := trace.SpanId
		traceId := trace.TraceId

		now := time.Now()
		transactionDuration := time.Duration(float64(transactionRange) * rand.Float64())
		transactionDelta := time.Duration(float64(transactionTimestampSpread) * rand.Float64())
		timestamp := now.Add(-transactionDelta)
		startTimestamp := timestamp.Add(-transactionDuration)

		retVal := Transaction{
			Timestamp:      toUtcString(timestamp),
			StartTimestamp: toUtcString(startTimestamp),
			EventId:        idGen(),
			Release:        relGen(),
			Transaction:    transGen(),
			Logger:         utils.SimpleRandomChoice([]string{"foo.bar.baz", "bam.baz.bad", ""}),
			Environment:    utils.SimpleRandomChoice([]string{"production", "development", "staging"}),
			User:           userGen(),
			Contexts: Contexts{
				Os:     osGen(),
				Device: deviceGen(),
				App:    appGen(),
				Trace:  trace,
			},
			Breadcrumbs:  breadcrumbsGen(),
			Measurements: measurementsGen(),
			Spans:        spansGen(transactionId, traceId, startTimestamp, timestamp),
		}

		return retVal
	}
}

func SpansGenerator(minSpans uint64, maxSpans uint64, operations []string) func(transactionId string, traceId string, transactionStart time.Time, timestamp time.Time) []Span {

	operationGen := OperationGenerator(operations)

	return func(transactionId string, traceId string, transactionStart time.Time, timestamp time.Time) []Span {
		numSpans := int(minSpans) + rand.Intn(int(maxSpans-minSpans))

		spans := make([]Span, 0, numSpans)
		childrenLeftFn := func() int64 { return rand.Int63n(3) + 1 } // something between 1 and 3

		numChildrenLeft := childrenLeftFn()
		currentNodeIdx := 0
		ts := toUnixTimestamp(timestamp)
		timeSlice := (ts - toUnixTimestamp(transactionStart)) / float64(numChildrenLeft)
		parentStart := toUnixTimestamp(transactionStart)
		parentId := transactionId

		for len(spans) < numSpans {
			if numChildrenLeft > 0 {
				startTimestamp := parentStart + timeSlice*float64(numChildrenLeft-1)
				ts = startTimestamp + timeSlice

				spans = append(spans, CreateSpan(parentId, traceId, ts, startTimestamp, operationGen()))

				numChildrenLeft -= 1
			} else {
				// decide how many sub spans for current span
				numChildrenLeft = childrenLeftFn()
				// divide the time for each sub-span equally
				currentNode := spans[currentNodeIdx]
				parentEnd := currentNode.Timestamp
				parentStart = currentNode.StartTimestamp
				parentId = currentNode.SpanId
				timeSlice = (parentEnd - parentStart) / float64(numChildrenLeft)
				currentNodeIdx += 1
			}
		}

		return spans
	}
}

func CreateSpan(parentId string, traceId string, timestamp float64, startTimestamp float64, operation string) Span {
	spanStatusGen := SpanStatusGenerator()
	spanIdGen := SpanIdGenerator()
	return Span{
		Status:         spanStatusGen(),
		Op:             operation,
		ParentSpanId:   parentId,
		SpanId:         spanIdGen(),
		TraceId:        traceId,
		Timestamp:      timestamp,
		StartTimestamp: startTimestamp,
	}
}

func OperationGenerator(operations []string) func() string {
	return func() string {
		return utils.SimpleRandomChoice(operations)
	}
}

func BreadcrumbsGenerator(min uint64, max uint64, categories []string, levels []string, types []string, messages []string) func() []Breadcrumb {
	if max == 0 {
		max = 50
	}
	if len(categories) == 0 {
		categories = []string{"auth", "web-request", "query"}
	}
	if len(levels) == 0 {
		levels = []string{"fatal", "error", "warning", "info", "debug"}
	}
	if len(types) == 0 {
		types = []string{"default", "http", "error"}
	}
	if len(messages) == 0 {
		messages = []string{
			"sending message via: UDP(10.8.0.10:53)",
			"GET http://localhost/xx/xxxx/xxxxxxxxxxxxxx [200]",
			"Authenticating the user_name",
			"IOError: [Errno 2] No such file or directory: '/tmp/someFile/'",
		}
	}

	return func() []Breadcrumb {
		numBreadcrumbs := int(min) + int(rand.Int63n(int64(max-min)))
		retVal := make([]Breadcrumb, 0, numBreadcrumbs)

		for idx := 0; idx < numBreadcrumbs; idx++ {
			retVal = append(retVal, Breadcrumb{
				Timestamp: toUnixTimestamp(time.Now()),
				Ty:        utils.SimpleRandomChoice(types),
				Category:  utils.SimpleRandomChoice(categories),
				Level:     utils.SimpleRandomChoice(levels),
				Message:   utils.SimpleRandomChoice(messages),
			})
		}

		return retVal
	}
}

func MeasurementsGenerator(measurements []string) func() map[string]float64 {
	return func() map[string]float64 {
		retVal := make(map[string]float64, len(measurements))
		for _, measurement := range measurements {
			retVal[measurement] = rand.Float64() * 1000
		}
		return retVal
	}
}

func DeviceContextGenerator() func() DeviceContext {
	return func() DeviceContext {

		if Flip() {
			return DeviceContext{}
		}

		screenResGen := func() string {
			if Flip() {
				return ""
			}
			return fmt.Sprintf("%dx%d", rand.Intn(1000), rand.Intn(1000))
		}

		nameGen := func() string {
			if Flip() {
				return ""
			}
			return fmt.Sprintf("Android SDK build for x%f", rand.Float32())
		}
		familyGen := func() string {
			if Flip() {
				return ""
			}
			return fmt.Sprintf("Device family %f", rand.Float32())
		}
		bootTimeGen := func() string {
			return fmt.Sprintf("%f", float64(time.Now().UnixNano())/1_000_000_000.0)
		}

		return DeviceContext{
			Type:                "device",
			Name:                nameGen(),
			Family:              familyGen(),
			Model:               "NYC-1",
			ModelId:             "NYC",
			Arch:                fmt.Sprintf("x%f", rand.Float32()),
			BatteryLevel:        rand.Float64() * 100,
			Orientation:         utils.SimpleRandomChoice([]string{"portrait", "landscape"}),
			Manufacturer:        utils.SimpleRandomChoice([]string{"Google", "Hasbro"}),
			Brand:               utils.SimpleRandomChoice([]string{"google", "zoogle", "moodle", "doodle", "tamagotchi"}),
			ScreenResolution:    screenResGen(),
			ScreenDensity:       uint64(rand.Int63n(5)),
			ScreenDpi:           uint64(rand.Int63n(1000)),
			Online:              Flip(),
			Charging:            Flip(),
			LowMemory:           Flip(),
			Simulator:           Flip(),
			MemorySize:          uint64(rand.Int63n(1000_000)),
			FreeMemory:          uint64(rand.Int63n(1000_000)),
			UsableMemory:        uint64(rand.Int63n(1000_000)),
			StorageSize:         uint64(rand.Int63n(1000_000)),
			FreeStorage:         uint64(rand.Int63n(1000_000)),
			ExternalStorageSize: uint64(rand.Int63n(1000_000)),
			ExternalFreeStorage: uint64(rand.Int63n(1000_000)),
			BootTime:            bootTimeGen(),
		}
	}
}

func AppContextGenerator() func() AppContext {
	return func() AppContext {
		if Flip() {
			return AppContext{}
		}
		appVersionGen := VersionGenerator(3, 10)

		return AppContext{
			Type:          "app",
			AppVersion:    appVersionGen(),
			AppIdentifier: "io.sentry.sample",
			AppName:       "sample",
			AppBuild:      appVersionGen(),
		}
	}
}

func SpanStatusGenerator() func() string {
	statuses := []string{"ok", "deadline_exceeded", "unauthenticated", "permission_denied", "not_found",
		"resource_exhausted", "invalid_argument", "unimplemented", "unavailable", "internal_error", "failure",
		"unknown", "cancelled", "already_exists", "failed_precondition", "aborted", "out_of_range", "data_loss"}

	return func() string {
		if rand.Int31n(101) < 100 {
			return "ok"
		}
		return utils.SimpleRandomChoice(statuses)
	}
}

func TraceContextGenerator(operations []string) func() TraceContext {
	operationGen := OperationGenerator(operations)
	return func() TraceContext {
		return TraceContext{
			Type:         "trace",
			TraceId:      EventIdGenerator()(),
			SpanId:       SpanIdGenerator()(),
			ParentSpanId: SpanIdGenerator()(),
			Op:           operationGen(),
			Status:       SpanStatusGenerator()(),
		}
	}
}

func OsContextGenerator() func() OsContext {
	verGen := VersionGenerator(3, 10)

	return func() OsContext {
		rooted := Flip()
		if Flip() {
			return OsContext{}
		} else {
			return OsContext{
				Type:          "os",
				Rooted:        &rooted,
				KernelVersion: "Linux version 3.10.0+ (bjoernj@bjoernj.mtv.corp.google.com) (gcc version 4.9.x 20150123 (prerelease) (GCC) ) #256 SMP PREEMPT Fri May 19 11:58:12 PDT 2017",
				Version:       verGen(),
				Built:         "sdk_google_phone_x86-userdebug 7.1.1 NYC 5464897 test-keys",
				Name:          utils.SimpleRandomChoice([]string{"Android", "NookPhone"}),
			}
		}
	}
}

func UserGenerator(maxUsers uint64) func() User {
	vg := VersionGenerator(4, 255)
	maxUsersInt := int64(maxUsers)
	return func() User {
		if maxUsers == 0 {
			return User{
				IpAddress: vg(),
			}
		}
		return User{
			IpAddress: vg(),
			Username:  fmt.Sprintf("Hobgoblin%f", rand.Float64()),
			Id:        fmt.Sprintf("%d", rand.Int63n(maxUsersInt)),
		}
	}
}

func VersionGenerator(numSegments uint64, maxValue uint64) func() string {
	return func() string {
		var buff bytes.Buffer

		for idx := uint64(1); idx <= numSegments; idx++ {
			buff.WriteString(fmt.Sprintf("%d", rand.Int63n(int64(maxValue))))
			if idx < numSegments {
				buff.WriteRune('.')
			}
		}
		return buff.String()
	}
}

func ReleaseGenerator(numReleases uint64) func() string {
	numRel := int64(numReleases)

	return func() string {
		if numRel == 0 {
			return ""
		}
		return fmt.Sprintf("release%d", rand.Int63n(numRel))
	}
}

func EventIdGenerator() func() string {
	return func() string {
		id := uuid.New()
		return utils.UuidAsHex(id)
	}
}

func SpanIdGenerator() func() string {
	return func() string {
		id := uuid.New()
		return utils.UuidAsHex(id)[0:16]
	}
}

// Flip returns a randomly generated bool (flips a coin)
func Flip() bool {
	if rand.Intn(2) == 0 {
		return false
	} else {
		return true
	}
}

func toUtcString(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}

func toUnixTimestamp(t time.Time) float64 {
	return float64(t.UnixNano()) / 1_000_000_000.0
}

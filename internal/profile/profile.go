package profile

import (
	"encoding/json"
	"time"

	"github.com/getsentry/vroom/internal/debugmeta"
	"github.com/getsentry/vroom/internal/metadata"
	"github.com/getsentry/vroom/internal/nodetree"
	"github.com/getsentry/vroom/internal/platform"
	"github.com/getsentry/vroom/internal/sample"
	"github.com/getsentry/vroom/internal/speedscope"
	"github.com/getsentry/vroom/internal/transaction"
)

type (
	profileInterface interface {
		GetDebugMeta() debugmeta.DebugMeta
		GetDurationNS() uint64
		GetEnvironment() string
		GetID() string
		GetOrganizationID() uint64
		GetPlatform() platform.Platform
		GetProjectID() uint64
		GetReceived() time.Time
		GetRelease() string
		GetRetentionDays() int
		GetTimestamp() time.Time
		GetTransaction() transaction.Transaction

		CallTrees() (map[uint64][]*nodetree.Node, error)
		Metadata() metadata.Metadata
		Normalize()
		Speedscope() (speedscope.Output, error)
		StoragePath() string
	}

	Profile struct {
		profile profileInterface
	}

	version struct {
		Version string `json:"version"`
	}
)

func New(p profileInterface) Profile {
	return Profile{
		profile: p,
	}
}

func (p *Profile) UnmarshalJSON(b []byte) error {
	var v version
	err := json.Unmarshal(b, &v)
	if err != nil {
		return err
	}
	switch v.Version {
	case "":
		p.profile = new(LegacyProfile)
	default:
		p.profile = new(sample.Profile)
	}
	return json.Unmarshal(b, &p.profile)
}

func (p Profile) MarshalJSON() ([]byte, error) {
	return json.Marshal(p.profile)
}

func (p *Profile) CallTrees() (map[uint64][]*nodetree.Node, error) {
	return p.profile.CallTrees()
}

func (p *Profile) DebugMeta() debugmeta.DebugMeta {
	return p.profile.GetDebugMeta()
}

func (p *Profile) ID() string {
	return p.profile.GetID()
}

func (p *Profile) OrganizationID() uint64 {
	return p.profile.GetOrganizationID()
}

func (p *Profile) ProjectID() uint64 {
	return p.profile.GetProjectID()
}

func (p *Profile) StoragePath() string {
	return p.profile.StoragePath()
}

func (p *Profile) Speedscope() (speedscope.Output, error) {
	return p.profile.Speedscope()
}

func (p *Profile) Metadata() metadata.Metadata {
	return p.profile.Metadata()
}

func (p *Profile) Platform() platform.Platform {
	return p.profile.GetPlatform()
}

func (p *Profile) Normalize() {
	p.profile.Normalize()
}

func (p *Profile) Transaction() transaction.Transaction {
	return p.profile.GetTransaction()
}

func (p *Profile) Environment() string {
	return p.profile.GetEnvironment()
}

func (p *Profile) Timestamp() time.Time {
	return p.profile.GetTimestamp()
}

func (p *Profile) Received() time.Time {
	return p.profile.GetReceived()
}

func (p *Profile) Release() string {
	return p.profile.GetRelease()
}

func (p *Profile) RetentionDays() int {
	return p.profile.GetRetentionDays()
}

func (p *Profile) DurationNS() uint64 {
	return p.profile.GetDurationNS()
}

package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"

	"cloud.google.com/go/storage"
	"github.com/getsentry/sentry-go"
	"github.com/getsentry/vroom/internal/chrometrace"
	"github.com/getsentry/vroom/internal/nodetree"
	"github.com/getsentry/vroom/internal/profile"
	"github.com/getsentry/vroom/internal/sample"
	"github.com/getsentry/vroom/internal/snubautil"
	"github.com/getsentry/vroom/internal/storageutil"
	"github.com/google/uuid"
	"github.com/julienschmidt/httprouter"
	"github.com/rs/zerolog/log"
	"google.golang.org/api/googleapi"
)

type PostProfileResponse struct {
	CallTrees map[uint64][]*nodetree.Node `json:"call_trees"`
}

type MinimalProfile struct {
	Platform string `json:"platform"`
	Version  string `json:"version"`
}

func (env *environment) postProfile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	hub := sentry.GetHubFromContext(ctx)

	s := sentry.StartSpan(ctx, "request.body")
	s.Description = "Read request body"
	body, err := io.ReadAll(r.Body)
	s.Finish()
	if err != nil {
		hub.CaptureException(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	s = sentry.StartSpan(ctx, "json.unmarshal")
	s.Description = "Unmarshal Snuba profile"
	var minimalProfile MinimalProfile
	err = json.Unmarshal(body, &minimalProfile)
	s.Finish()
	if err != nil {
		log.Err(err).Msg("minimal profile can't be unmarshaled")
		hub.CaptureException(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var p profile.Profile

	// if it's a sample format
	if len(minimalProfile.Version) > 0 {
		var sampleProfile sample.Profile
		err = json.Unmarshal(body, &sampleProfile)
		s.Finish()
		if err != nil {
			log.Err(err).Str("profile", string(body)).Msg("profile can't be unmarshaled")
			hub.CaptureException(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		p = &sampleProfile
	} else {
		var legacyProfile profile.LegacyProfile
		err = json.Unmarshal(body, &legacyProfile)
		s.Finish()
		if err != nil {
			log.Err(err).Msg("profile can't be unmarshaled")
			hub.CaptureException(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		p = &legacyProfile
	}

	hub.Scope().SetTags(map[string]string{
		"organization_id": strconv.FormatUint(p.GetOrganizationID(), 10),
		"project_id":      strconv.FormatUint(p.GetProjectID(), 10),
		"profile_id":      p.GetID(),
	})

	s = sentry.StartSpan(ctx, "gcs.write")
	s.Description = "Write profile to GCS"
	_, err = storageutil.CompressedWrite(ctx, env.profilesBucket, p.StoragePath(), p)
	s.Finish()
	if err != nil {
		hub.CaptureException(err)
		var e *googleapi.Error
		if ok := errors.As(err, &e); ok {
			w.WriteHeader(http.StatusBadGateway)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	}

	s = sentry.StartSpan(ctx, "calltree")
	s.Description = "Generate call trees"
	callTrees, _ := p.CallTrees()
	s.Finish()

	s = sentry.StartSpan(ctx, "json.marshal")
	s.Description = "Marshal call trees"
	defer s.Finish()

	b, err := json.Marshal(PostProfileResponse{
		CallTrees: callTrees,
	})
	if err != nil {
		hub.CaptureException(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(b)
}

func getRawProfile(ctx context.Context, organizationID, projectID uint64, profileID string, profilesBucket *storage.BucketHandle, snuba snubautil.Client) (profile.LegacyProfile, error) {
	var p profile.LegacyProfile
	err := storageutil.UnmarshalCompressed(ctx, profilesBucket, profile.StoragePath(organizationID, projectID, profileID), &p)
	if err != nil {
		if !errors.Is(err, storage.ErrObjectNotExist) {
			return profile.LegacyProfile{}, err
		}
		sqb, err := snuba.NewQuery(ctx, "profiles")
		if err != nil {
			return profile.LegacyProfile{}, err
		}
		sp, err := snubautil.GetProfile(organizationID, projectID, profileID, sqb)
		if err != nil {
			return profile.LegacyProfile{}, err
		}
		p = sp
	}

	return p, nil
}

func (env *environment) getRawProfile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	hub := sentry.GetHubFromContext(ctx)
	ps := httprouter.ParamsFromContext(ctx)
	rawOrganizationID := ps.ByName("organization_id")
	organizationID, err := strconv.ParseUint(rawOrganizationID, 10, 64)
	if err != nil {
		hub.CaptureException(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	hub.Scope().SetTag("organization_id", rawOrganizationID)

	rawProjectID := ps.ByName("project_id")
	projectID, err := strconv.ParseUint(rawProjectID, 10, 64)
	if err != nil {
		sentry.CaptureException(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	hub.Scope().SetTag("project_id", rawProjectID)

	profileID := ps.ByName("profile_id")
	_, err = uuid.Parse(profileID)
	if err != nil {
		hub.CaptureException(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	hub.Scope().SetTag("profile_id", profileID)
	s := sentry.StartSpan(ctx, "profile.read")
	s.Description = "Read profile from GCS or Snuba"

	p, err := getRawProfile(ctx, organizationID, projectID, profileID, env.profilesBucket, env.snuba)
	s.Finish()
	if err != nil {
		if errors.Is(err, snubautil.ErrProfileNotFound) {
			w.WriteHeader(http.StatusNotFound)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	}

	s = sentry.StartSpan(ctx, "json.marshal")
	defer s.Finish()
	b, err := json.Marshal(p)
	if err != nil {
		hub.CaptureException(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=3600, immutable")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(b)
}

func (env *environment) getProfile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	hub := sentry.GetHubFromContext(ctx)
	ps := httprouter.ParamsFromContext(ctx)
	rawOrganizationID := ps.ByName("organization_id")
	organizationID, err := strconv.ParseUint(rawOrganizationID, 10, 64)
	if err != nil {
		hub.CaptureException(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	hub.Scope().SetTag("organization_id", rawOrganizationID)

	rawProjectID := ps.ByName("project_id")
	projectID, err := strconv.ParseUint(rawProjectID, 10, 64)
	if err != nil {
		sentry.CaptureException(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	hub.Scope().SetTag("project_id", rawProjectID)

	profileID := ps.ByName("profile_id")
	_, err = uuid.Parse(profileID)
	if err != nil {
		hub.CaptureException(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	hub.Scope().SetTag("profile_id", profileID)
	s := sentry.StartSpan(ctx, "profile.read")
	s.Description = "Read profile from GCS or Snuba"

	p, err := getRawProfile(ctx, organizationID, projectID, profileID, env.profilesBucket, env.snuba)
	s.Finish()
	if err != nil {
		if errors.Is(err, snubautil.ErrProfileNotFound) {
			w.WriteHeader(http.StatusNotFound)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	}

	hub.Scope().SetTag("platform", p.Platform)

	s = sentry.StartSpan(ctx, "json.marshal")
	defer s.Finish()

	var b []byte
	switch p.Platform {
	case "typescript", "javascript":
		b = p.Profile
	default:
		b, err = chrometrace.SpeedscopeFromSnuba(p)
	}
	if err != nil {
		hub.CaptureException(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=3600, immutable")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(b)
}

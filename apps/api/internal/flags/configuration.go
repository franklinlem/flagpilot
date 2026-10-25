package flags

import (
	"context"
	"fmt"
	"github.com/franklinlem/flagpilot/apps/api/internal/auth"
	"github.com/franklinlem/flagpilot/apps/api/internal/identifiers"
	"github.com/franklinlem/flagpilot/apps/api/internal/pagination"
	"github.com/franklinlem/flagpilot/apps/api/internal/projects"
	q "github.com/franklinlem/flagpilot/apps/api/internal/storage/postgres/sqlc"
	"github.com/jackc/pgx/v5/pgtype"
)

func environmentScope(ctx context.Context, qt *q.Queries, user, id pgtype.UUID, write bool) (q.Environment, q.Project, string, error) {
	env, err := qt.ScopedFlagEnvironment(ctx, q.ScopedFlagEnvironmentParams{ID: id, UserID: user})
	if err != nil {
		return env, q.Project{}, "", storageError(err)
	}
	permission := auth.ReadFlags
	if write {
		permission = auth.ManageFlags
	}
	project, role, err := projects.Authorize(ctx, qt, user, env.ProjectID, permission, false)
	if err != nil {
		return env, project, role, err
	}
	// Escrita de configuração não bloqueia projeto: somente o ambiente afetado.
	if write {
		env, err = qt.WriteFlagEnvironment(ctx, id)
	} else {
		env, err = qt.ReadFlagEnvironment(ctx, id)
	}
	if err != nil {
		return env, project, role, storageError(err)
	}
	return env, project, role, nil
}
func toConfig(flagID, envID pgtype.UUID, key string, enabled bool, rollout int32, version, revision int64) Config {
	return Config{FlagID: identifiers.String(flagID), FlagKey: key, EnvironmentID: identifiers.String(envID), Enabled: enabled, RolloutBps: rollout, Version: version, Revision: revision}
}
func toEnvironment(row q.Environment) projects.Environment {
	return projects.Environment{ID: identifiers.String(row.ID), ProjectID: identifiers.String(row.ProjectID), Name: row.Name, Slug: row.Slug, IsProduction: row.IsProduction, Revision: row.Revision}
}
func (s *Service) Configs(ctx context.Context, actor, environment string, page pagination.PageRequest) (Snapshot, string, error) {
	out := Snapshot{Items: []Config{}}
	etag := ""
	user, id, err := parse(actor, environment)
	if err != nil {
		return out, etag, err
	}
	err = s.transaction(ctx, func(ctx context.Context, qt *q.Queries) error {
		env, project, role, err := environmentScope(ctx, qt, user, id, false)
		if err != nil {
			return err
		}
		scope := fmt.Sprintf("%s:%s:%d", identifiers.String(id), identifiers.String(user), env.Revision)
		after, err := page.After("configs", scope)
		if err != nil {
			return err
		}
		rows, err := qt.ListFlagConfigs(ctx, q.ListFlagConfigsParams{EnvironmentID: id, AfterID: after, PageLimit: int32(page.Limit + 1)})
		if err != nil {
			return ErrUnavailable
		}
		if len(rows) > page.Limit {
			out.NextCursor = pagination.NextCursor("configs", scope, rows[page.Limit-1].FlagID)
			rows = rows[:page.Limit]
		}
		for _, row := range rows {
			out.Items = append(out.Items, toConfig(row.FlagID, id, row.FlagKey, row.Enabled, row.RolloutBps, row.Version, env.Revision))
		}
		out.Environment = toEnvironment(env)
		out.ProjectName = project.Name
		out.Role = role
		etag = EnvironmentETag(identifiers.String(id), env.Revision, role)
		return nil
	})
	return out, etag, err
}
func (s *Service) GetConfig(ctx context.Context, actor, environment, key string) (Config, string, error) {
	var out Config
	etag := ""
	user, id, err := parse(actor, environment)
	if err != nil {
		return out, etag, err
	}
	if !keyPattern.MatchString(key) {
		return out, etag, ErrValidation
	}
	err = s.transaction(ctx, func(ctx context.Context, qt *q.Queries) error {
		env, _, role, err := environmentScope(ctx, qt, user, id, false)
		if err != nil {
			return err
		}
		row, err := qt.FlagConfigByKey(ctx, q.FlagConfigByKeyParams{EnvironmentID: id, Key: key})
		if err != nil {
			return storageError(err)
		}
		out = toConfig(row.FlagID, id, row.FlagKey, row.Enabled, row.RolloutBps, row.Version, env.Revision)
		etag = EnvironmentETag(identifiers.String(id), env.Revision, role)
		return nil
	})
	return out, etag, err
}
func (s *Service) PutConfig(ctx context.Context, actor, environment, key, match string, enabled bool, rollout int32, confirmProduction bool) (Config, string, error) {
	var out Config
	etag := ""
	user, id, err := parse(actor, environment)
	if err != nil {
		return out, etag, err
	}
	if !keyPattern.MatchString(key) || (rollout != 0 && rollout != 10000) {
		return out, etag, ErrValidation
	}
	err = s.transaction(ctx, func(ctx context.Context, qt *q.Queries) error {
		env, project, role, err := environmentScope(ctx, qt, user, id, true)
		if err != nil {
			return err
		}
		row, err := qt.FlagConfigByKey(ctx, q.FlagConfigByKeyParams{EnvironmentID: id, Key: key})
		if err != nil {
			return storageError(err)
		}
		if err = precondition(match, EnvironmentETag(identifiers.String(id), env.Revision, role)); err != nil {
			return err
		}
		if env.IsProduction && !confirmProduction {
			return ErrProduction
		}
		before := toConfig(row.FlagID, id, key, row.Enabled, row.RolloutBps, row.Version, env.Revision)
		if err = qt.ChangeFlagConfig(ctx, q.ChangeFlagConfigParams{EnvironmentID: id, FlagID: row.FlagID, Enabled: enabled, RolloutBps: rollout}); err != nil {
			return ErrUnavailable
		}
		revision, err := qt.BumpEnvironment(ctx, id)
		if err != nil {
			return ErrUnavailable
		}
		out = toConfig(row.FlagID, id, key, enabled, rollout, row.Version+1, revision)
		etag = EnvironmentETag(identifiers.String(id), revision, role)
		return audit(ctx, qt, project.OrganizationID, user, row.ID, "flag.config_updated", "flag_config", before, out)
	})
	return out, etag, err
}

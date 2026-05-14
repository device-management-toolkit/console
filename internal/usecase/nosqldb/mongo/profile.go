package mongo

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/device-management-toolkit/console/internal/entity"
	"github.com/device-management-toolkit/console/internal/usecase/profiles"
	"github.com/device-management-toolkit/console/pkg/logger"
)

type ProfileRepo struct {
	col          *mongo.Collection
	ieee8021xCol *mongo.Collection
	log          logger.Interface
}

var _ profiles.Repository = (*ProfileRepo)(nil)

func NewProfileRepo(db *mongo.Database, log logger.Interface) *ProfileRepo {
	return &ProfileRepo{
		col:          db.Collection(CollectionProfiles),
		ieee8021xCol: db.Collection(CollectionIEEE8021xConfigs),
		log:          log,
	}
}

func (r *ProfileRepo) GetCount(ctx context.Context, tenantID string) (int, error) {
	if tenantID != "" && !identifierRegex.MatchString(tenantID) {
		return 0, nil
	}

	n, err := r.col.CountDocuments(ctx, bson.M{fieldTenantID: tenantID})
	if err != nil {
		return 0, errProfileDatabase.Wrap("GetCount", "CountDocuments", err)
	}

	return int(n), nil
}

func (r *ProfileRepo) Get(ctx context.Context, top, skip int, tenantID string) ([]entity.Profile, error) {
	if tenantID != "" && !identifierRegex.MatchString(tenantID) {
		return []entity.Profile{}, nil
	}

	limit := int64(DefaultTop)
	if top > 0 {
		limit = int64(top)
	}

	offset := int64(0)
	if skip > 0 {
		offset = int64(skip)
	}

	cur, err := r.col.Find(ctx, bson.M{fieldTenantID: tenantID},
		options.Find().SetSort(bson.D{{Key: fieldProfileName, Value: 1}}).SetLimit(limit).SetSkip(offset))
	if err != nil {
		return nil, errProfileDatabase.Wrap("Get", "Find", err)
	}
	defer cur.Close(ctx)

	out := make([]entity.Profile, 0)
	if err := cur.All(ctx, &out); err != nil {
		return nil, errProfileDatabase.Wrap("Get", "Cursor.All", err)
	}

	// SQL joins ieee8021xconfigs; do it per row in code.
	for i := range out {
		if err := r.populate8021x(ctx, &out[i]); err != nil {
			return nil, err
		}
	}

	return out, nil
}

func (r *ProfileRepo) GetByName(ctx context.Context, profileName, tenantID string) (*entity.Profile, error) {
	if !identifierRegex.MatchString(profileName) {
		return nil, nil
	}

	if tenantID != "" && !identifierRegex.MatchString(tenantID) {
		return nil, nil
	}

	p := entity.Profile{}

	err := r.col.FindOne(ctx, bson.M{fieldProfileName: profileName, fieldTenantID: tenantID}).Decode(&p)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}

		return nil, errProfileDatabase.Wrap("GetByName", "FindOne", err)
	}

	if err := r.populate8021x(ctx, &p); err != nil {
		return nil, err
	}

	return &p, nil
}

// populate8021x mirrors the SQL LEFT JOIN — missing row → nil join fields, driver error propagates.
func (r *ProfileRepo) populate8021x(ctx context.Context, p *entity.Profile) error {
	if p.IEEE8021xProfileName == nil || *p.IEEE8021xProfileName == "" {
		return nil
	}

	var ieee entity.IEEE8021xConfig
	if err := r.ieee8021xCol.FindOne(ctx, bson.M{
		fieldProfileName: *p.IEEE8021xProfileName,
		fieldTenantID:    p.TenantID,
	}).Decode(&ieee); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil
		}

		return errProfileDatabase.Wrap("populate8021x", "FindOne", err)
	}

	ap := ieee.AuthenticationProtocol
	p.AuthenticationProtocol = &ap
	p.PXETimeout = ieee.PXETimeout

	wi := ieee.WiredInterface
	p.WiredInterface = &wi

	return nil
}

func (r *ProfileRepo) Delete(ctx context.Context, profileName, tenantID string) (bool, error) {
	if !identifierRegex.MatchString(profileName) {
		return false, nil
	}

	if tenantID != "" && !identifierRegex.MatchString(tenantID) {
		return false, nil
	}

	res, err := r.col.DeleteOne(ctx, bson.M{fieldProfileName: profileName, fieldTenantID: tenantID})
	if err != nil {
		return false, errProfileDatabase.Wrap("Delete", "DeleteOne", err)
	}

	return res.DeletedCount > 0, nil
}

func (r *ProfileRepo) Update(ctx context.Context, p *entity.Profile) (bool, error) {
	if !identifierRegex.MatchString(p.ProfileName) {
		return false, errProfileDatabase.Wrap("Update", "validate", nil)
	}

	if p.TenantID != "" && !identifierRegex.MatchString(p.TenantID) {
		return false, errProfileDatabase.Wrap("Update", "validate", nil)
	}

	// profile_name and tenant_id are filter keys, intentionally omitted from $set.
	set := bson.M{
		"activation":                 p.Activation,
		"amtpassword":                p.AMTPassword,
		"generaterandompassword":     p.GenerateRandomPassword,
		"ciraconfigname":             nullIfEmptyPtr(p.CIRAConfigName),
		"mebxpassword":               p.MEBXPassword,
		"generaterandommebxpassword": p.GenerateRandomMEBxPassword,
		fieldTags:                    p.Tags,
		"dhcpenabled":                p.DHCPEnabled,
		"tlsmode":                    p.TLSMode,
		"userconsent":                p.UserConsent,
		"iderenabled":                p.IDEREnabled,
		"kvmenabled":                 p.KVMEnabled,
		"solenabled":                 p.SOLEnabled,
		"tlssigningauthority":        p.TLSSigningAuthority,
		fieldIEEE8021xProfileName:    nullIfEmptyPtr(p.IEEE8021xProfileName),
		"ipsyncenabled":              p.IPSyncEnabled,
		"localwifisyncenabled":       p.LocalWiFiSyncEnabled,
		"uefiwifisyncenabled":        p.UEFIWiFiSyncEnabled,
	}

	res, err := r.col.UpdateOne(ctx,
		bson.M{fieldProfileName: p.ProfileName, fieldTenantID: p.TenantID},
		bson.M{opSet: set},
	)
	if err != nil {
		return false, errProfileDatabase.Wrap("Update", "UpdateOne", err)
	}

	return res.MatchedCount > 0, nil
}

func (r *ProfileRepo) Insert(ctx context.Context, p *entity.Profile) (string, error) {
	if !identifierRegex.MatchString(p.ProfileName) {
		return "", errProfileDatabase.Wrap("Insert", "validate", nil)
	}

	if p.TenantID != "" && !identifierRegex.MatchString(p.TenantID) {
		return "", errProfileDatabase.Wrap("Insert", "validate", nil)
	}

	// Empty-string pointers → nil → BSON null (matches SQL NULL-on-empty).
	toInsert := *p
	toInsert.CIRAConfigName = nullIfEmptyPtr(p.CIRAConfigName)
	toInsert.IEEE8021xProfileName = nullIfEmptyPtr(p.IEEE8021xProfileName)

	if _, err := r.col.InsertOne(ctx, toInsert); err != nil {
		if isDuplicateKey(err) {
			return "", errProfileNotUnique.Wrap(err.Error())
		}

		return "", errProfileDatabase.Wrap("Insert", "InsertOne", err)
	}

	return "", nil
}

// nullIfEmptyPtr returns nil for (nil || pointer to "") so BSON stores null.
func nullIfEmptyPtr(s *string) *string {
	if s == nil || *s == "" {
		return nil
	}

	return s
}

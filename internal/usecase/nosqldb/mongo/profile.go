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
	n, err := r.col.CountDocuments(ctx, bson.M{"tenantid": tenantID})
	if err != nil {
		return 0, errProfileDatabase.Wrap("GetCount", "CountDocuments", err)
	}

	return int(n), nil
}

func (r *ProfileRepo) Get(ctx context.Context, top, skip int, tenantID string) ([]entity.Profile, error) {
	const defaultTop = 100

	limit := int64(defaultTop)
	if top > 0 {
		limit = int64(top)
	}

	offset := int64(0)
	if skip > 0 {
		offset = int64(skip)
	}

	cur, err := r.col.Find(ctx, bson.M{"tenantid": tenantID},
		options.Find().SetSort(bson.D{{Key: "profilename", Value: 1}}).SetLimit(limit).SetSkip(offset))
	if err != nil {
		return nil, errProfileDatabase.Wrap("Get", "Find", err)
	}
	defer cur.Close(ctx)

	out := make([]entity.Profile, 0)
	if err := cur.All(ctx, &out); err != nil {
		return nil, errProfileDatabase.Wrap("Get", "Cursor.All", err)
	}

	// SQL version joins the 8021x config for auth_protocol / pxe_timeout / wired_interface.
	// Do the join per row in code.
	for i := range out {
		r.populate8021x(ctx, &out[i])
	}

	return out, nil
}

func (r *ProfileRepo) GetByName(ctx context.Context, profileName, tenantID string) (*entity.Profile, error) {
	p := entity.Profile{}

	err := r.col.FindOne(ctx, bson.M{"profilename": profileName, "tenantid": tenantID}).Decode(&p)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}

		return nil, errProfileDatabase.Wrap("GetByName", "FindOne", err)
	}

	r.populate8021x(ctx, &p)

	return &p, nil
}

// populate8021x fills the join-result fields on Profile from the matching
// ieee8021xconfigs document, if any. Mirrors the LEFT JOIN in the SQL path.
func (r *ProfileRepo) populate8021x(ctx context.Context, p *entity.Profile) {
	if p.IEEE8021xProfileName == nil || *p.IEEE8021xProfileName == "" {
		return
	}

	var ieee entity.IEEE8021xConfig
	if err := r.ieee8021xCol.FindOne(ctx, bson.M{
		"profilename": *p.IEEE8021xProfileName,
		"tenantid":    p.TenantID,
	}).Decode(&ieee); err != nil {
		if !errors.Is(err, mongo.ErrNoDocuments) {
			r.log.Warn("populate8021x: %v", err)
		}

		return
	}

	ap := ieee.AuthenticationProtocol
	p.AuthenticationProtocol = &ap
	p.PXETimeout = ieee.PXETimeout

	wi := ieee.WiredInterface
	p.WiredInterface = &wi
}

func (r *ProfileRepo) Delete(ctx context.Context, profileName, tenantID string) (bool, error) {
	res, err := r.col.DeleteOne(ctx, bson.M{"profilename": profileName, "tenantid": tenantID})
	if err != nil {
		return false, errProfileDatabase.Wrap("Delete", "DeleteOne", err)
	}

	return res.DeletedCount > 0, nil
}

func (r *ProfileRepo) Update(ctx context.Context, p *entity.Profile) (bool, error) {
	// Same field list the SQL UPDATE sets — leaving profile_name / tenant_id
	// out of the $set, since those are the filter.
	set := bson.M{
		"activation":                 p.Activation,
		"amtpassword":                p.AMTPassword,
		"generaterandompassword":     p.GenerateRandomPassword,
		"ciraconfigname":             nullIfEmptyPtr(p.CIRAConfigName),
		"mebxpassword":               p.MEBXPassword,
		"generaterandommebxpassword": p.GenerateRandomMEBxPassword,
		"tags":                       p.Tags,
		"dhcpenabled":                p.DHCPEnabled,
		"tlsmode":                    p.TLSMode,
		"userconsent":                p.UserConsent,
		"iderenabled":                p.IDEREnabled,
		"kvmenabled":                 p.KVMEnabled,
		"solenabled":                 p.SOLEnabled,
		"tlssigningauthority":        p.TLSSigningAuthority,
		"ieee8021xprofilename":       nullIfEmptyPtr(p.IEEE8021xProfileName),
		"ipsyncenabled":              p.IPSyncEnabled,
		"localwifisyncenabled":       p.LocalWiFiSyncEnabled,
		"uefiwifisyncenabled":        p.UEFIWiFiSyncEnabled,
	}

	res, err := r.col.UpdateOne(ctx,
		bson.M{"profilename": p.ProfileName, "tenantid": p.TenantID},
		bson.M{"$set": set},
	)
	if err != nil {
		return false, errProfileDatabase.Wrap("Update", "UpdateOne", err)
	}

	return res.MatchedCount > 0, nil
}

func (r *ProfileRepo) Insert(ctx context.Context, p *entity.Profile) (string, error) {
	// Normalise empty-string pointers to nil so they serialize as BSON null,
	// matching the SQL path's NULL-on-empty behavior.
	cira := nullIfEmptyPtr(p.CIRAConfigName)
	ieee := nullIfEmptyPtr(p.IEEE8021xProfileName)

	doc := bson.M{
		"profilename":                p.ProfileName,
		"activation":                 p.Activation,
		"amtpassword":                p.AMTPassword,
		"generaterandompassword":     p.GenerateRandomPassword,
		"ciraconfigname":             cira,
		"mebxpassword":               p.MEBXPassword,
		"generaterandommebxpassword": p.GenerateRandomMEBxPassword,
		"tags":                       p.Tags,
		"dhcpenabled":                p.DHCPEnabled,
		"tlsmode":                    p.TLSMode,
		"userconsent":                p.UserConsent,
		"iderenabled":                p.IDEREnabled,
		"kvmenabled":                 p.KVMEnabled,
		"solenabled":                 p.SOLEnabled,
		"tlssigningauthority":        p.TLSSigningAuthority,
		"ieee8021xprofilename":       ieee,
		"ipsyncenabled":              p.IPSyncEnabled,
		"localwifisyncenabled":       p.LocalWiFiSyncEnabled,
		"tenantid":                   p.TenantID,
		"uefiwifisyncenabled":        p.UEFIWiFiSyncEnabled,
	}

	if _, err := r.col.InsertOne(ctx, doc); err != nil {
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

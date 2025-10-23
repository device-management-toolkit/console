package openapi

import (
	"github.com/go-fuego/fuego"

	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
)

func (f *FuegoAdapter) RegisterProfileRoutes() {
	fuego.Get(f.server, "/api/v1/admin/profiles", f.getProfiles,
		fuego.OptionTags("Profiles"),
		fuego.OptionSummary("List Profiles"),
		fuego.OptionDescription("Retrieve all profiles with optional pagination"),
		fuego.OptionQueryInt("$top", "Number of records to return"),
		fuego.OptionQueryInt("$skip", "Number of records to skip"),
		fuego.OptionQueryBool("$count", "Include total count"),
	)

	fuego.Get(f.server, "/api/v1/admin/profiles/{name}", f.getProfileByName,
		fuego.OptionTags("Profiles"),
		fuego.OptionSummary("Get Profile by Name"),
		fuego.OptionDescription("Retrieve a specific profile by name"),
		fuego.OptionPath("name", "Profile name"),
	)

	fuego.Post(f.server, "/api/v1/admin/profiles", f.createProfile,
		fuego.OptionTags("Profiles"),
		fuego.OptionSummary("Create Profile"),
		fuego.OptionDescription("Create a new profile"),
	)

	fuego.Patch(f.server, "/api/v1/admin/profiles", f.updateProfile,
		fuego.OptionTags("Profiles"),
		fuego.OptionSummary("Update Profile"),
		fuego.OptionDescription("Update an existing profile"),
	)

	fuego.Delete(f.server, "/api/v1/admin/profiles/{name}", f.deleteProfile,
		fuego.OptionTags("Profiles"),
		fuego.OptionSummary("Delete Profile"),
		fuego.OptionDescription("Delete a profile by name"),
		fuego.OptionPath("name", "Profile name"),
	)

	fuego.Get(f.server, "/api/v1/admin/profiles/export/{name}", f.exportProfile,
		fuego.OptionTags("Profiles"),
		fuego.OptionSummary("Export Profile"),
		fuego.OptionDescription("Export a profile configuration"),
		fuego.OptionPath("name", "Profile name"),
		fuego.OptionQuery("domainName", "Domain name for export"),
	)
}

func (f *FuegoAdapter) getProfiles(_ fuego.ContextNoBody) (dto.ProfileCountResponse, error) {
	profiles := []dto.Profile{
		{
			ProfileName:                "example-profile",
			AMTPassword:                "Password123!",
			GenerateRandomPassword:     false,
			Activation:                 "ccmactivate",
			MEBXPassword:               "MEBXPass123!",
			GenerateRandomMEBxPassword: false,
			DHCPEnabled:                true,
			IPSyncEnabled:              true,
			LocalWiFiSyncEnabled:       true,
			TenantID:                   "default",
			TLSMode:                    1,
			TLSSigningAuthority:        "SelfSigned",
			UserConsent:                "All",
			IDEREnabled:                true,
		},
	}

	return dto.ProfileCountResponse{
		Count: len(profiles),
		Data:  profiles,
	}, nil
}

func (f *FuegoAdapter) getProfileByName(_ fuego.ContextNoBody) (dto.Profile, error) {
	return dto.Profile{
		ProfileName:                "example-profile",
		AMTPassword:                "Password123!",
		GenerateRandomPassword:     false,
		Activation:                 "ccmactivate",
		MEBXPassword:               "MEBXPass123!",
		GenerateRandomMEBxPassword: false,
		DHCPEnabled:                true,
		IPSyncEnabled:              true,
		LocalWiFiSyncEnabled:       true,
		TenantID:                   "default",
		TLSMode:                    1,
		TLSSigningAuthority:        "SelfSigned",
		UserConsent:                "All",
		IDEREnabled:                true,
	}, nil
}

func (f *FuegoAdapter) createProfile(c fuego.ContextWithBody[dto.Profile]) (dto.Profile, error) {
	body, err := c.Body()
	if err != nil {
		return dto.Profile{}, err
	}

	return body, nil
}

func (f *FuegoAdapter) updateProfile(c fuego.ContextWithBody[dto.Profile]) (dto.Profile, error) {
	body, err := c.Body()
	if err != nil {
		return dto.Profile{}, err
	}

	return body, nil
}

func (f *FuegoAdapter) deleteProfile(_ fuego.ContextNoBody) (any, error) {
	return nil, nil
}

func (f *FuegoAdapter) exportProfile(_ fuego.ContextNoBody) (dto.ProfileExportResponse, error) {
	return dto.ProfileExportResponse{
		Filename: "example-profile.yaml",
		Content:  "# Example profile YAML content",
		Key:      "example-key",
	}, nil
}

package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
	"github.com/spf13/cobra"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/hayfordstanley/eightfivetwo/services/api/internal/adapter/mongostore"
	"github.com/hayfordstanley/eightfivetwo/services/api/internal/domain"
)

const imageFolder = "eightfivetwo"

var errCloudinaryNotConfigured = errors.New("cloudinary is not configured (set CLOUDINARY_* in services/api/.env)")

// seedImage maps a local file to a Cloudinary public-id name and, optionally,
// the design slug it should be attached to as the cover photo.
type seedImage struct {
	file       string // basename within --dir
	publicID   string // public-id name (folder is prefixed)
	designSlug string // "" for the standalone hero image
}

func imageManifest() []seedImage {
	return []seedImage{
		{file: "hero.png", publicID: "hero-atelier", designSlug: ""},
		{file: "blazer.png", publicID: "boardroom-blazer", designSlug: "boardroom-blazer"},
		{file: "gown.png", publicID: "osu-gown", designSlug: "osu-gown"},
		{file: "linen.png", publicID: "linen-weekend-set", designSlug: "linen-weekend-set"},
		{file: "trousers.png", publicID: "cigarette-trousers", designSlug: "tailored-cigarette-trousers"},
		{file: "wrap.png", publicID: "sika-wrap-dress", designSlug: "sika-wrap-dress"},
	}
}

func newSeedImagesCommand() *cobra.Command {
	var dir string

	cmd := &cobra.Command{
		Use:   "seed-images",
		Short: "Upload local editorial images to Cloudinary and attach them to designs",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := loadEnvironment()
			if err != nil {
				return err
			}

			if !cfg.UploadsEnabled() {
				return errCloudinaryNotConfigured
			}

			cld, err := cloudinary.NewFromParams(
				cfg.CloudinaryCloudName, cfg.CloudinaryAPIKey, cfg.CloudinaryAPISecret)
			if err != nil {
				return fmt.Errorf("cloudinary client: %w", err)
			}

			return withDatabase(cmd.Context(), cfg, func(db *mongo.Database) error {
				return runSeedImages(cmd.Context(), cmd.OutOrStdout(), cld, db, dir)
			})
		},
	}
	cmd.Flags().StringVar(&dir, "dir", "/tmp/e25img", "directory holding the local image files")

	return cmd
}

func runSeedImages(
	ctx context.Context, out io.Writer, cld *cloudinary.Cloudinary, db *mongo.Database, dir string,
) error {
	designs := mongostore.NewDesignRepository(db)
	overwrite := true

	for _, img := range imageManifest() {
		result, err := cld.Upload.Upload(ctx, filepath.Join(dir, img.file), uploader.UploadParams{
			PublicID:  img.publicID,
			Folder:    imageFolder,
			Overwrite: &overwrite,
		})
		if err != nil {
			return fmt.Errorf("upload %s: %w", img.file, err)
		}

		_, _ = fmt.Fprintf(out, "uploaded %s -> %s\n", img.file, result.PublicID)

		if img.designSlug == "" {
			_, _ = fmt.Fprintf(out, "  (hero image — reference publicId %q in the web app)\n", result.PublicID)

			continue
		}

		err = attachPhoto(ctx, designs, img.designSlug, result.PublicID)
		if err != nil {
			return err
		}

		_, _ = fmt.Fprintf(out, "  attached to design /%s\n", img.designSlug)
	}

	return nil
}

func attachPhoto(ctx context.Context, designs *mongostore.DesignRepository, slug, publicID string) error {
	design, err := designs.GetBySlug(ctx, slug)
	if err != nil {
		return fmt.Errorf("load design %s: %w", slug, err)
	}

	design.Photos = []domain.Photo{{PublicID: publicID, Order: 0}}

	err = designs.Update(ctx, design)
	if err != nil {
		return fmt.Errorf("attach photo to %s: %w", slug, err)
	}

	return nil
}

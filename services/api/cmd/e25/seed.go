package main

import (
	"context"
	"fmt"
	"io"

	"github.com/spf13/cobra"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/hayfordstanley/eightfivetwo/services/api/internal/adapter/mongostore"
	"github.com/hayfordstanley/eightfivetwo/services/api/internal/domain"
	"github.com/hayfordstanley/eightfivetwo/services/api/internal/service"
)

const seedDepositPesewas = 500_00 // GHS 500, the scope's starting deposit

type seedCollection struct {
	name    string
	note    string
	designs []service.DesignInput
}

func newSeedCommand() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "seed",
		Short: "Seed demo collections, designs, and store settings",
		Long: "Seeds the database with demo catalog content and store settings. " +
			"Skips seeding when collections already exist unless --force is given.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := loadEnvironment()
			if err != nil {
				return err
			}

			return withDatabase(cmd.Context(), cfg, func(db *mongo.Database) error {
				return runSeed(cmd.Context(), cmd.OutOrStdout(), db, force)
			})
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "seed even when collections already exist")

	return cmd
}

func runSeed(ctx context.Context, out io.Writer, db *mongo.Database, force bool) error {
	collections := mongostore.NewCollectionRepository(db)
	designs := mongostore.NewDesignRepository(db)
	settings := mongostore.NewSettingsRepository(db)

	for _, ensure := range []interface {
		EnsureIndexes(ctx context.Context) error
	}{collections, designs} {
		err := ensure.EnsureIndexes(ctx)
		if err != nil {
			return fmt.Errorf("ensure indexes: %w", err)
		}
	}

	existing, err := collections.List(ctx, true)
	if err != nil {
		return fmt.Errorf("check existing collections: %w", err)
	}

	if len(existing) > 0 && !force {
		_, _ = fmt.Fprintf(out, "skipping: %d collection(s) already exist (use --force to seed anyway)\n", len(existing))

		return seedSettings(ctx, out, settings)
	}

	catalog := service.NewCatalog(collections, designs)

	for _, seed := range seedCatalog() {
		collection, err := catalog.CreateCollection(ctx, seed.name, seed.note)
		if err != nil {
			return fmt.Errorf("seed collection %q: %w", seed.name, err)
		}

		for _, input := range seed.designs {
			input.CollectionID = collection.ID

			design, err := catalog.CreateDesign(ctx, input)
			if err != nil {
				return fmt.Errorf("seed design %q: %w", input.Name, err)
			}

			_, _ = fmt.Fprintf(out, "seeded %s / %s (/designs/%s)\n", collection.Name, design.Name, design.Slug)
		}
	}

	return seedSettings(ctx, out, settings)
}

func seedSettings(ctx context.Context, out io.Writer, repo domain.SettingsRepository) error {
	store := service.NewStoreSettings(repo)

	err := store.Update(ctx, &domain.Settings{
		DepositPesewas: seedDepositPesewas,
		WhatsAppNumber: "+233200000000",
		VisitLocation:  "Osu, Accra",
		DeliveryRates: []domain.DeliveryRate{
			{Area: "Accra", RatePesewas: 30_00},
			{Area: "Tema", RatePesewas: 50_00},
			{Area: "Kumasi", RatePesewas: 80_00},
		},
	})
	if err != nil {
		return fmt.Errorf("seed settings: %w", err)
	}

	_, _ = fmt.Fprintln(out, "seeded store settings (deposit GHS 500, 3 delivery areas)")

	return nil
}

func seedCatalog() []seedCollection {
	chart := func(bust, waist, hip string) map[string]string {
		return map[string]string{"bust": bust, "waist": waist, "hip": hip}
	}
	bands := func(base int64) []domain.SizeBand {
		return []domain.SizeBand{
			{Label: "8", PricePesewas: base, Chart: chart("86 cm", "66 cm", "92 cm")},
			{Label: "10", PricePesewas: base, Chart: chart("90 cm", "70 cm", "96 cm")},
			{Label: "12", PricePesewas: base + 50_00, Chart: chart("94 cm", "74 cm", "100 cm")},
			{Label: "14", PricePesewas: base + 50_00, Chart: chart("98 cm", "78 cm", "104 cm")},
		}
	}

	return []seedCollection{
		{
			name: "The Boardroom Edit",
			note: "Sharp tailoring for the office — the pieces that carry a working week.",
			designs: []service.DesignInput{
				{Name: "Boardroom Blazer", Note: "Single-breasted, structured shoulder.", SizeBands: bands(850_00)},
				{Name: "Tailored Cigarette Trousers", Note: "High waist, cropped at the ankle.", SizeBands: bands(450_00)},
				{Name: "The Power Shift Dress", Note: "Knee-length shift with a sharp seam line.", SizeBands: bands(620_00)},
			},
		},
		{
			name: "Accra Nights",
			note: "Evening pieces cut for movement — limited bolt of midnight crepe.",
			designs: []service.DesignInput{
				{Name: "Osu Gown", Note: "Floor-length, open back.", SizeBands: bands(1200_00)},
				{Name: "Sika Wrap Dress", Note: "Wrap silhouette in midnight crepe.", SizeBands: bands(780_00)},
			},
		},
		{
			name: "Harmattan Capsule",
			note: "Light layers for the dry season.",
			designs: []service.DesignInput{
				{Name: "Linen Weekend Set", Note: "Two-piece relaxed set.", SizeBands: bands(540_00)},
				{Name: "Sahel Shirt Dress", Note: "Breathable shirt dress with belt.", SizeBands: bands(490_00)},
			},
		},
	}
}

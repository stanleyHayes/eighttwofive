package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/hayfordstanley/eightfivetwo/services/api/internal/adapter/mongostore"
	"github.com/hayfordstanley/eightfivetwo/services/api/internal/domain"
	"github.com/hayfordstanley/eightfivetwo/services/api/internal/service"
)

const (
	targetDesigns     = 38
	targetSubscribers = 30
	photosPerDesign   = 3
)

// imagePool returns Cloudinary public IDs confirmed to render. Seeded designs
// reuse these so demo images are never broken, even when several designs share
// the same underlying asset.
func imagePool() []string {
	return []string{
		"eightfivetwo/osu-gown",
		"eightfivetwo/sika-wrap-dress",
		"eightfivetwo/linen-weekend-set",
		"eightfivetwo/cigarette-trousers",
		"eightfivetwo/boardroom-blazer",
		"eightfivetwo/hero-atelier",
	}
}

type designSpec struct {
	collection string
	name       string
	note       string
	base       int64
}

func newSeedMoreCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "seed-more",
		Short: "Add more demo designs, photos, and subscribers (reusing valid images)",
		Long: "Backfills every existing design with multiple rendering photos, grows the " +
			"catalogue to ~" + strconv.Itoa(targetDesigns) + " designs so admin pagination is " +
			"visible, and seeds ~" + strconv.Itoa(targetSubscribers) + " subscribers. Converges " +
			"to those targets, so it is safe to run more than once.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := loadEnvironment()
			if err != nil {
				return err
			}

			return withDatabase(cmd.Context(), cfg, func(db *mongo.Database) error {
				return runSeedMore(cmd.Context(), cmd.OutOrStdout(), db)
			})
		},
	}
}

// designPhotos builds photosPerDesign photos starting at a rotating offset, so
// different designs surface different covers while always using valid assets.
func designPhotos(offset int) []domain.Photo {
	pool := imagePool()

	doubled := make([]string, 0, len(pool)*2)
	doubled = append(doubled, pool...)
	doubled = append(doubled, pool...)

	skip := offset % len(pool)
	order := 0
	out := make([]domain.Photo, 0, photosPerDesign)

	for _, publicID := range doubled {
		if skip > 0 {
			skip--

			continue
		}

		if order >= photosPerDesign {
			break
		}

		out = append(out, domain.Photo{PublicID: publicID, Order: order})
		order++
	}

	return out
}

func seedBands(base int64) []domain.SizeBand {
	chart := func(bust, waist, hip string) map[string]string {
		return map[string]string{"bust": bust, "waist": waist, "hip": hip}
	}

	return []domain.SizeBand{
		{Label: "8", PricePesewas: base, Chart: chart("86 cm", "66 cm", "92 cm")},
		{Label: "10", PricePesewas: base, Chart: chart("90 cm", "70 cm", "96 cm")},
		{Label: "12", PricePesewas: base + 50_00, Chart: chart("94 cm", "74 cm", "100 cm")},
		{Label: "14", PricePesewas: base + 50_00, Chart: chart("98 cm", "78 cm", "104 cm")},
	}
}

func runSeedMore(ctx context.Context, out io.Writer, db *mongo.Database) error {
	collections := mongostore.NewCollectionRepository(db)
	designs := mongostore.NewDesignRepository(db)
	subscribers := mongostore.NewSubscriberRepository(db)

	err := designs.EnsureIndexes(ctx)
	if err != nil {
		return fmt.Errorf("ensure design indexes: %w", err)
	}

	err = subscribers.EnsureIndexes(ctx)
	if err != nil {
		return fmt.Errorf("ensure subscriber indexes: %w", err)
	}

	err = enrichExistingPhotos(ctx, out, designs)
	if err != nil {
		return err
	}

	err = growCatalogue(ctx, out, collections, designs)
	if err != nil {
		return err
	}

	return seedSubscribers(ctx, out, subscribers)
}

// enrichExistingPhotos gives every existing design a full set of rendering
// photos, so none shows a broken cover and "set as main" is meaningful.
func enrichExistingPhotos(ctx context.Context, out io.Writer, designs *mongostore.DesignRepository) error {
	all, err := designs.List(ctx, domain.DesignFilter{})
	if err != nil {
		return fmt.Errorf("list designs: %w", err)
	}

	for idx := range all {
		design := all[idx]
		design.Photos = designPhotos(idx)

		err = designs.Update(ctx, &design)
		if err != nil {
			return fmt.Errorf("backfill photos for %s: %w", design.Slug, err)
		}
	}

	_, _ = fmt.Fprintf(out, "backfilled photos on %d existing design(s)\n", len(all))

	return nil
}

// growCatalogue creates new designs until the total reaches targetDesigns, so
// the admin designs list spans more than one page.
func growCatalogue(
	ctx context.Context,
	out io.Writer,
	collections *mongostore.CollectionRepository,
	designs *mongostore.DesignRepository,
) error {
	current, err := designs.Count(ctx, domain.DesignFilter{})
	if err != nil {
		return fmt.Errorf("count designs: %w", err)
	}

	needed := targetDesigns - int(current)
	if needed <= 0 {
		_, _ = fmt.Fprintf(out, "catalogue already has %d designs — skipping new designs\n", current)

		return nil
	}

	cols, err := collections.List(ctx, false)
	if err != nil {
		return fmt.Errorf("list collections: %w", err)
	}

	byName := make(map[string]string, len(cols))
	for _, col := range cols {
		byName[col.Name] = col.ID
	}

	catalog := service.NewCatalog(collections, designs)
	created := 0

	for idx, spec := range newDesignSpecs() {
		if created >= needed {
			break
		}

		collectionID, ok := byName[spec.collection]
		if !ok {
			continue
		}

		design, err := catalog.CreateDesign(ctx, service.DesignInput{
			CollectionID: collectionID,
			Name:         spec.name,
			Note:         spec.note,
			Photos:       designPhotos(idx + 1),
			SizeBands:    seedBands(spec.base),
		})
		if err != nil {
			return fmt.Errorf("create design %q: %w", spec.name, err)
		}

		created++

		_, _ = fmt.Fprintf(out, "seeded %s / %s (/designs/%s)\n", spec.collection, design.Name, design.Slug)
	}

	_, _ = fmt.Fprintf(out, "created %d new design(s) — catalogue now ~%d\n", created, int(current)+created)

	return nil
}

func seedSubscribers(ctx context.Context, out io.Writer, repo *mongostore.SubscriberRepository) error {
	firsts := []string{"Ama", "Akua", "Esi", "Yaa", "Abena", "Adwoa", "Afia", "Akosua", "Efua", "Maame"}
	lasts := []string{"Mensah", "Owusu", "Boateng", "Asante", "Addo", "Sarpong", "Annan", "Quaye", "Darko", "Frimpong"}

	now := time.Now().UTC()
	made := 0
	seq := 0

	for _, last := range lasts {
		if seq >= targetSubscribers {
			break
		}

		for _, first := range firsts {
			if seq >= targetSubscribers {
				break
			}

			email := fmt.Sprintf("%s.%s%02d@example.com", strings.ToLower(first), strings.ToLower(last), seq+1)
			sub := &domain.Subscriber{
				Email:     email,
				Name:      first + " " + last,
				CreatedAt: now.Add(-time.Duration(seq) * 30 * time.Hour),
			}

			err := repo.Create(ctx, sub)
			seq++

			if errors.Is(err, domain.ErrDuplicateEmail) {
				continue
			}

			if err != nil {
				return fmt.Errorf("create subscriber %s: %w", email, err)
			}

			made++
		}
	}

	_, _ = fmt.Fprintf(out, "seeded %d new subscriber(s) (target %d)\n", made, targetSubscribers)

	return nil
}

func newDesignSpecs() []designSpec {
	specs := boardroomSpecs()
	specs = append(specs, accraSpecs()...)
	specs = append(specs, harmattanSpecs()...)

	return specs
}

func boardroomSpecs() []designSpec {
	const c = "The Boardroom Edit"

	return []designSpec{
		{collection: c, name: "Pinstripe Trouser Suit", note: "Two-piece in fine wool pinstripe.", base: 980_00},
		{collection: c, name: "Double-Breasted Blazer", note: "Peak lapel, gold buttons.", base: 820_00},
		{collection: c, name: "Pencil Skirt", note: "High-waisted, back vent.", base: 360_00},
		{collection: c, name: "Sheath Dress", note: "Darted sheath with cap sleeve.", base: 640_00},
		{collection: c, name: "Wide-Leg Suit Trouser", note: "Pressed crease, full break.", base: 480_00},
		{collection: c, name: "Waistcoat & Trouser Set", note: "Tailored waistcoat over slim trouser.", base: 740_00},
		{collection: c, name: "Tailored Wrap Coat", note: "Belted wrap coat in melton.", base: 1100_00},
		{collection: c, name: "Crepe Work Blouse", note: "Tie-neck crepe blouse.", base: 320_00},
		{collection: c, name: "Pleated Midi Skirt", note: "Knife pleats, satin-backed.", base: 420_00},
		{collection: c, name: "Belted Blazer Dress", note: "Double-breasted blazer dress.", base: 760_00},
		{collection: c, name: "Structured Shift", note: "Clean shift with bracelet sleeve.", base: 580_00},
	}
}

func accraSpecs() []designSpec {
	const c = "Accra Nights"

	return []designSpec{
		{collection: c, name: "Midnight Slip Dress", note: "Bias-cut slip in midnight crepe.", base: 720_00},
		{collection: c, name: "Crepe Column Gown", note: "Floor-length column, side slit.", base: 1240_00},
		{collection: c, name: "Asymmetric Drape Dress", note: "One-shoulder draped bodice.", base: 880_00},
		{collection: c, name: "Off-Shoulder Gown", note: "Sweetheart neckline, full skirt.", base: 1180_00},
		{collection: c, name: "Beaded Cocktail Dress", note: "Hand-beaded mini.", base: 940_00},
		{collection: c, name: "Satin Wrap Gown", note: "Wrap front in heavy satin.", base: 1020_00},
		{collection: c, name: "High-Slit Evening Dress", note: "Thigh-high slit, halter neck.", base: 860_00},
		{collection: c, name: "Cowl-Back Gown", note: "Draped cowl back, fluid skirt.", base: 1150_00},
		{collection: c, name: "Velvet Tuxedo Dress", note: "Tuxedo-collar velvet mini.", base: 900_00},
		{collection: c, name: "Halter Maxi", note: "Open back, sweeping hem.", base: 980_00},
	}
}

func harmattanSpecs() []designSpec {
	const c = "Harmattan Capsule"

	return []designSpec{
		{collection: c, name: "Linen Shirt Dress", note: "Belted linen shirt dress.", base: 520_00},
		{collection: c, name: "Cotton Wrap Skirt", note: "Midi wrap in crisp cotton.", base: 300_00},
		{collection: c, name: "Relaxed Linen Trouser", note: "Drawstring waist, tapered leg.", base: 360_00},
		{collection: c, name: "Gauze Maxi Dress", note: "Airy double-gauze maxi.", base: 620_00},
		{collection: c, name: "Poplin Sundress", note: "Tiered poplin with smocked back.", base: 480_00},
		{collection: c, name: "Linen Jumpsuit", note: "Wide-leg linen jumpsuit.", base: 700_00},
		{collection: c, name: "Breezy Kaftan", note: "Embroidered yoke kaftan.", base: 540_00},
		{collection: c, name: "Cropped Linen Set", note: "Cropped shirt and short set.", base: 580_00},
		{collection: c, name: "Tiered Cotton Dress", note: "Three-tier cotton midi.", base: 500_00},
		{collection: c, name: "Wide Linen Culottes", note: "Cropped culottes, side pockets.", base: 340_00},
	}
}

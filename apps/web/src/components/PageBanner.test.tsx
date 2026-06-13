import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import StorefrontOutlined from "@mui/icons-material/StorefrontOutlined";
import { PageBanner } from "./PageBanner";

function renderBanner() {
  return render(
    <MemoryRouter>
      <PageBanner
        title="The Store"
        description="Browse the live collections and limited runs."
        breadcrumbs={[
          { label: "Home", to: "/" },
          { label: "Store" },
        ]}
        icon={<StorefrontOutlined />}
      />
    </MemoryRouter>,
  );
}

describe("PageBanner", () => {
  it("renders the title as the page heading", () => {
    renderBanner();
    expect(screen.getByRole("heading", { level: 1, name: "The Store" })).toBeInTheDocument();
  });

  it("renders the description", () => {
    renderBanner();
    expect(
      screen.getByText("Browse the live collections and limited runs."),
    ).toBeInTheDocument();
  });

  it("links breadcrumbs with a `to` and renders the current page as plain text", () => {
    renderBanner();
    const home = screen.getByRole("link", { name: "Home" });
    expect(home).toHaveAttribute("href", "/");
    // The current (last) crumb is not a link.
    expect(screen.queryByRole("link", { name: "Store" })).not.toBeInTheDocument();
    expect(screen.getByText("Store")).toBeInTheDocument();
  });

  it("renders an action button linking to a route", () => {
    render(
      <MemoryRouter>
        <PageBanner
          title="Designs"
          breadcrumbs={[{ label: "Admin", to: "/admin" }, { label: "Designs" }]}
          icon={<StorefrontOutlined />}
          action={{ to: "/admin/designs/new", label: "New design" }}
        />
      </MemoryRouter>,
    );
    expect(screen.getByRole("link", { name: "New design" })).toHaveAttribute(
      "href",
      "/admin/designs/new",
    );
  });
});

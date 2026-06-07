package symbols

import (
	"testing"

	"github.com/codesharpdev/beakon/pkg"
)

// React/TS declaration forms the extractor must cover: arrow-assigned
// components, HOC-wrapped (memo/forwardRef) components, and default-exported
// functions. Regression corpus from dogfooding on a real React Router app.
const tsReactSrc = `
import { memo } from "react";

export const VersionDropdown = memo(function VersionDropdown(props) {
  return renderDropdown(props);
});

const Sidebar = () => {
  return buildSidebar();
};

export default function CommentComposer(props) {
  return composeComment(props);
}
`

func TestExtractTS_ConstArrowComponent(t *testing.T) {
	nodes, _ := Extract("app/sidebar.tsx", "typescript", []byte(tsReactSrc))
	assertContains(t, symbolNames(nodes), "Sidebar")
}

func TestExtractTS_MemoWrappedComponent(t *testing.T) {
	nodes, _ := Extract("app/version-dropdown.tsx", "typescript", []byte(tsReactSrc))
	assertContains(t, symbolNames(nodes), "VersionDropdown")
}

func TestExtractTS_DefaultExportFunction(t *testing.T) {
	nodes, _ := Extract("app/comment-composer.tsx", "typescript", []byte(tsReactSrc))
	assertContains(t, symbolNames(nodes), "CommentComposer")
}

func TestExtractTS_ReactCallEdges(t *testing.T) {
	_, calls := Extract("app/sidebar.tsx", "typescript", []byte(tsReactSrc))
	assertEdge(t, calls, "Sidebar", "buildSidebar")
	assertEdge(t, calls, "VersionDropdown", "renderDropdown")
	assertEdge(t, calls, "CommentComposer", "composeComment")
}

// A .tsx file with JSX must parse with the JSX-aware grammar; with the plain
// TypeScript grammar the JSX subtrees become ERROR nodes and the file yields
// nothing.
const tsxSrc = `
function Toolbar() {
  return <button onClick={handleClick}>x</button>;
}

export default function Page() {
  return (
    <Layout>
      <Toolbar />
      <Sidebar open={true} />
      <Menu.Item />
    </Layout>
  );
}
`

func TestExtractTSX_ParsesJSX(t *testing.T) {
	nodes, _ := Extract("app/page.tsx", "typescript", []byte(tsxSrc))
	names := symbolNames(nodes)
	assertContains(t, names, "Page")
	assertContains(t, names, "Toolbar")
}

func TestExtractTSX_JSXAsCallEdge(t *testing.T) {
	_, calls := Extract("app/page.tsx", "typescript", []byte(tsxSrc))
	assertEdge(t, calls, "Page", "Toolbar")
	assertEdge(t, calls, "Page", "Sidebar")
	assertEdge(t, calls, "Page", "Layout")
	assertEdge(t, calls, "Page", "Menu.Item")
	// Host elements are not symbols — must not become edges.
	assertNoEdge(t, calls, "Toolbar", "button")
}

func assertNoEdge(t *testing.T, calls []pkg.CallEdge, from, to string) {
	t.Helper()
	for _, c := range calls {
		if c.From == from && c.To == to {
			t.Errorf("unexpected call edge %q → %q", from, to)
			return
		}
	}
}

package symbols

import (
	"testing"
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

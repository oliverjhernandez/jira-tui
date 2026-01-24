package main

func (m model) getPanelWidth() int {
	return max(120, m.windowWidth-4)
}

func (m model) getContentWidth() int {
	return m.getPanelWidth() - 6
}

func (m model) getPanelHeight() int {
	infoPanelHeight := 6
	return m.windowHeight - 2 - infoPanelHeight
}

func (m model) getModalWidth(scale float64) int {
	return int(float64(m.windowWidth) * scale)
}

func (m model) getModalHeight(scale float64) int {
	return int(float64(m.windowHeight) * scale)
}

func (m model) getSmallModalWidth() int {
	return m.getModalWidth(0.4)
}

func (m model) getMediumModalWidth() int {
	return m.getModalWidth(0.6)
}

func (m model) getLargeModalWidth() int {
	return m.getModalWidth(0.7)
}

func (m model) getListViewportWidth() int {
	return m.windowWidth - 4
}

func (m model) getListViewportHeight() int {
	infoPanelHeight := 6
	return m.windowHeight - 3 - infoPanelHeight
}

func (m model) getDetailViewportWidth() int {
	return m.windowWidth - 10
}

func (m model) getDetailViewportHeight() int {
	headerHeight := 15
	footerHeight := 1
	return m.windowHeight - headerHeight - footerHeight
}

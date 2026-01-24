package main

func (m model) getPanelWidth() int {
	return max(120, m.windowWidth-4)
}

func (m model) getContentWidth() int {
	return m.getPanelWidth() - 6
}

func (m model) getPanelHeight() int {
	infoPanelHeight := 5
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

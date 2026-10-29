package market

func (m *marketManager) ForRun(runID string) (MarketManager, error) {
	scoped, err := m.st.ForRun(runID)
	if err != nil {
		return nil, err
	}
	run := *m
	run.st = scoped
	return &run, nil
}

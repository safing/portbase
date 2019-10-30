package updater

// Export exports the list of resources. All resources must be locked when accessed.
func (reg *ResourceRegistry) Export() map[string]*Resource {
	reg.RLock()
	defer reg.RUnlock()

	// copy the map
	new := make(map[string]*Resource)
	for key, val := range reg.resources {
		new[key] = val
	}

	return new
}

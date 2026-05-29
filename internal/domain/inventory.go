package domain

import (
	"errors"
	"strings"
)

var (
	ErrMissingCIID       = errors.New("ci id is required")
	ErrDuplicateCIID     = errors.New("ci id already exists")
	ErrComponentNotFound = errors.New("component not found")
)

type Inventory struct {
	components map[string]Component
	order      []string
}

func NewInventory() *Inventory {
	return &Inventory{
		components: make(map[string]Component),
	}
}

func (i *Inventory) Add(component Component) error {
	ciID := strings.TrimSpace(component.CIID)
	if ciID == "" {
		return ErrMissingCIID
	}
	if err := component.Validate(); err != nil {
		return err
	}
	i.ensureInitialized()
	if _, exists := i.components[ciID]; exists {
		return ErrDuplicateCIID
	}

	component.CIID = ciID
	i.components[ciID] = component
	i.order = append(i.order, ciID)
	return nil
}

func (i *Inventory) List() []Component {
	components := make([]Component, 0, len(i.order))
	for _, id := range i.order {
		component, exists := i.components[id]
		if exists {
			components = append(components, component)
		}
	}
	return components
}

func (i *Inventory) Get(ciID string) (Component, error) {
	component, exists := i.components[strings.TrimSpace(ciID)]
	if !exists {
		return Component{}, ErrComponentNotFound
	}
	return component, nil
}

func (i *Inventory) Remove(ciID string) error {
	ciID = strings.TrimSpace(ciID)
	if _, exists := i.components[ciID]; !exists {
		return ErrComponentNotFound
	}

	delete(i.components, ciID)
	for index, existingID := range i.order {
		if existingID == ciID {
			i.order = append(i.order[:index], i.order[index+1:]...)
			break
		}
	}
	return nil
}

func (i *Inventory) ensureInitialized() {
	if i.components == nil {
		i.components = make(map[string]Component)
	}
}

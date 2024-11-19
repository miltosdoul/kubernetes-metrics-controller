package client

const NoSystemFieldSelector = "metadata.namespace!=kube-system"

// ClientWrapperInterface is a custom client interface used to separate concerns
// TODO: add podlister, podListerSynced to interface type
type ClientWrapperInterface interface {
	// UpdateList retrieves all the resources of this type using the K8s client
	UpdateList()
	// List lists all items which are stored for this type
	List() *[]any
	// ListWithRefresh refreshes local list and retrieves
	ListWithRefresh() *[]any
}

package file

// BySize implements sort.Interface for sorting a list of files by size.
type BySize []File

// ByPath implements sort.Interface for sorting a list of files by path.
type ByPath []File

func (a BySize) Len() int           { return len(a) }
func (a ByPath) Len() int           { return len(a) }
func (a BySize) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByPath) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a BySize) Less(i, j int) bool { return a[i].Size < a[j].Size }

// Sort by root path and then by filename within each root.
func (a ByPath) Less(i, j int) bool {
	if a[i].Root == a[j].Root {
		return a[i].Name < a[j].Name
	}
	return a[i].Root < a[j].Root
}

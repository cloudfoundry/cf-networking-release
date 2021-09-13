package bindmount

type Mounter struct{}

func (m *Mounter) IdempotentlyMount(source, target string) error {
	return nil
}

func (m *Mounter) RemoveMount(target string) error {
	return nil
}

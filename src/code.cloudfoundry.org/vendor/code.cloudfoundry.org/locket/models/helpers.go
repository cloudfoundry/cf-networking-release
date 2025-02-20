package models

func GetResource(resource *Resource) *Resource {
	r := &Resource{Key: resource.Key, Owner: resource.Owner, Value: resource.Value}
	if resource.TypeCode == TypeCode_UNKNOWN {
		r.TypeCode = GetTypeCode(resource.Type)
		r.Type = resource.Type
	} else {
		r.TypeCode = resource.TypeCode
		r.Type = GetType(resource)
	}
	return r
}

func GetTypeCode(lockType string) TypeCode {
	switch lockType {
	case LockType:
		return TypeCode_LOCK
	case PresenceType:
		return TypeCode_PRESENCE
	default:
		return TypeCode_UNKNOWN
	}
}

func GetType(resource *Resource) string {
	switch resource.TypeCode {
	case TypeCode_LOCK:
		return LockType
	case TypeCode_PRESENCE:
		return PresenceType
	default:
		return resource.Type
	}
}

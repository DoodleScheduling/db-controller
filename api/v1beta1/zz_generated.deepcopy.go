//go:build !ignore_autogenerated

/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Code generated by controller-gen. DO NOT EDIT.

package v1beta1

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DatabaseReference) DeepCopyInto(out *DatabaseReference) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DatabaseReference.
func (in *DatabaseReference) DeepCopy() *DatabaseReference {
	if in == nil {
		return nil
	}
	out := new(DatabaseReference)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DatabaseSpec) DeepCopyInto(out *DatabaseSpec) {
	*out = *in
	if in.Timeout != nil {
		in, out := &in.Timeout, &out.Timeout
		*out = new(v1.Duration)
		**out = **in
	}
	if in.RootSecret != nil {
		in, out := &in.RootSecret, &out.RootSecret
		*out = new(SecretReference)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DatabaseSpec.
func (in *DatabaseSpec) DeepCopy() *DatabaseSpec {
	if in == nil {
		return nil
	}
	out := new(DatabaseSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Extension) DeepCopyInto(out *Extension) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Extension.
func (in *Extension) DeepCopy() *Extension {
	if in == nil {
		return nil
	}
	out := new(Extension)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in Extensions) DeepCopyInto(out *Extensions) {
	{
		in := &in
		*out = make(Extensions, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Extensions.
func (in Extensions) DeepCopy() Extensions {
	if in == nil {
		return nil
	}
	out := new(Extensions)
	in.DeepCopyInto(out)
	return *out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Grant) DeepCopyInto(out *Grant) {
	*out = *in
	if in.Privileges != nil {
		in, out := &in.Privileges, &out.Privileges
		*out = make([]Privilege, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Grant.
func (in *Grant) DeepCopy() *Grant {
	if in == nil {
		return nil
	}
	out := new(Grant)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MongoDBDatabase) DeepCopyInto(out *MongoDBDatabase) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MongoDBDatabase.
func (in *MongoDBDatabase) DeepCopy() *MongoDBDatabase {
	if in == nil {
		return nil
	}
	out := new(MongoDBDatabase)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *MongoDBDatabase) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MongoDBDatabaseList) DeepCopyInto(out *MongoDBDatabaseList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]MongoDBDatabase, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MongoDBDatabaseList.
func (in *MongoDBDatabaseList) DeepCopy() *MongoDBDatabaseList {
	if in == nil {
		return nil
	}
	out := new(MongoDBDatabaseList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *MongoDBDatabaseList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MongoDBDatabaseSpec) DeepCopyInto(out *MongoDBDatabaseSpec) {
	*out = *in
	if in.DatabaseSpec != nil {
		in, out := &in.DatabaseSpec, &out.DatabaseSpec
		*out = new(DatabaseSpec)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MongoDBDatabaseSpec.
func (in *MongoDBDatabaseSpec) DeepCopy() *MongoDBDatabaseSpec {
	if in == nil {
		return nil
	}
	out := new(MongoDBDatabaseSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MongoDBDatabaseStatus) DeepCopyInto(out *MongoDBDatabaseStatus) {
	*out = *in
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]v1.Condition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MongoDBDatabaseStatus.
func (in *MongoDBDatabaseStatus) DeepCopy() *MongoDBDatabaseStatus {
	if in == nil {
		return nil
	}
	out := new(MongoDBDatabaseStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MongoDBUser) DeepCopyInto(out *MongoDBUser) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MongoDBUser.
func (in *MongoDBUser) DeepCopy() *MongoDBUser {
	if in == nil {
		return nil
	}
	out := new(MongoDBUser)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *MongoDBUser) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MongoDBUserList) DeepCopyInto(out *MongoDBUserList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]MongoDBUser, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MongoDBUserList.
func (in *MongoDBUserList) DeepCopy() *MongoDBUserList {
	if in == nil {
		return nil
	}
	out := new(MongoDBUserList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *MongoDBUserList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MongoDBUserRole) DeepCopyInto(out *MongoDBUserRole) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MongoDBUserRole.
func (in *MongoDBUserRole) DeepCopy() *MongoDBUserRole {
	if in == nil {
		return nil
	}
	out := new(MongoDBUserRole)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MongoDBUserSpec) DeepCopyInto(out *MongoDBUserSpec) {
	*out = *in
	if in.Database != nil {
		in, out := &in.Database, &out.Database
		*out = new(DatabaseReference)
		**out = **in
	}
	if in.Credentials != nil {
		in, out := &in.Credentials, &out.Credentials
		*out = new(SecretReference)
		**out = **in
	}
	if in.Roles != nil {
		in, out := &in.Roles, &out.Roles
		*out = new([]MongoDBUserRole)
		if **in != nil {
			in, out := *in, *out
			*out = make([]MongoDBUserRole, len(*in))
			copy(*out, *in)
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MongoDBUserSpec.
func (in *MongoDBUserSpec) DeepCopy() *MongoDBUserSpec {
	if in == nil {
		return nil
	}
	out := new(MongoDBUserSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MongoDBUserStatus) DeepCopyInto(out *MongoDBUserStatus) {
	*out = *in
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]v1.Condition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MongoDBUserStatus.
func (in *MongoDBUserStatus) DeepCopy() *MongoDBUserStatus {
	if in == nil {
		return nil
	}
	out := new(MongoDBUserStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PostgreSQLDatabase) DeepCopyInto(out *PostgreSQLDatabase) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PostgreSQLDatabase.
func (in *PostgreSQLDatabase) DeepCopy() *PostgreSQLDatabase {
	if in == nil {
		return nil
	}
	out := new(PostgreSQLDatabase)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *PostgreSQLDatabase) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PostgreSQLDatabaseList) DeepCopyInto(out *PostgreSQLDatabaseList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]PostgreSQLDatabase, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PostgreSQLDatabaseList.
func (in *PostgreSQLDatabaseList) DeepCopy() *PostgreSQLDatabaseList {
	if in == nil {
		return nil
	}
	out := new(PostgreSQLDatabaseList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *PostgreSQLDatabaseList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PostgreSQLDatabaseSpec) DeepCopyInto(out *PostgreSQLDatabaseSpec) {
	*out = *in
	if in.DatabaseSpec != nil {
		in, out := &in.DatabaseSpec, &out.DatabaseSpec
		*out = new(DatabaseSpec)
		(*in).DeepCopyInto(*out)
	}
	if in.Extensions != nil {
		in, out := &in.Extensions, &out.Extensions
		*out = make(Extensions, len(*in))
		copy(*out, *in)
	}
	if in.SearchPath != nil {
		in, out := &in.SearchPath, &out.SearchPath
		*out = make(Schemas, len(*in))
		copy(*out, *in)
	}
	if in.Schemas != nil {
		in, out := &in.Schemas, &out.Schemas
		*out = make(Schemas, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PostgreSQLDatabaseSpec.
func (in *PostgreSQLDatabaseSpec) DeepCopy() *PostgreSQLDatabaseSpec {
	if in == nil {
		return nil
	}
	out := new(PostgreSQLDatabaseSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PostgreSQLDatabaseStatus) DeepCopyInto(out *PostgreSQLDatabaseStatus) {
	*out = *in
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]v1.Condition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PostgreSQLDatabaseStatus.
func (in *PostgreSQLDatabaseStatus) DeepCopy() *PostgreSQLDatabaseStatus {
	if in == nil {
		return nil
	}
	out := new(PostgreSQLDatabaseStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PostgreSQLUser) DeepCopyInto(out *PostgreSQLUser) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PostgreSQLUser.
func (in *PostgreSQLUser) DeepCopy() *PostgreSQLUser {
	if in == nil {
		return nil
	}
	out := new(PostgreSQLUser)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *PostgreSQLUser) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PostgreSQLUserList) DeepCopyInto(out *PostgreSQLUserList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]PostgreSQLUser, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PostgreSQLUserList.
func (in *PostgreSQLUserList) DeepCopy() *PostgreSQLUserList {
	if in == nil {
		return nil
	}
	out := new(PostgreSQLUserList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *PostgreSQLUserList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PostgreSQLUserSpec) DeepCopyInto(out *PostgreSQLUserSpec) {
	*out = *in
	if in.Database != nil {
		in, out := &in.Database, &out.Database
		*out = new(DatabaseReference)
		**out = **in
	}
	if in.Credentials != nil {
		in, out := &in.Credentials, &out.Credentials
		*out = new(SecretReference)
		**out = **in
	}
	if in.Grants != nil {
		in, out := &in.Grants, &out.Grants
		*out = make([]Grant, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Roles != nil {
		in, out := &in.Roles, &out.Roles
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PostgreSQLUserSpec.
func (in *PostgreSQLUserSpec) DeepCopy() *PostgreSQLUserSpec {
	if in == nil {
		return nil
	}
	out := new(PostgreSQLUserSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PostgreSQLUserStatus) DeepCopyInto(out *PostgreSQLUserStatus) {
	*out = *in
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]v1.Condition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PostgreSQLUserStatus.
func (in *PostgreSQLUserStatus) DeepCopy() *PostgreSQLUserStatus {
	if in == nil {
		return nil
	}
	out := new(PostgreSQLUserStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Schema) DeepCopyInto(out *Schema) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Schema.
func (in *Schema) DeepCopy() *Schema {
	if in == nil {
		return nil
	}
	out := new(Schema)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in Schemas) DeepCopyInto(out *Schemas) {
	{
		in := &in
		*out = make(Schemas, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Schemas.
func (in Schemas) DeepCopy() Schemas {
	if in == nil {
		return nil
	}
	out := new(Schemas)
	in.DeepCopyInto(out)
	return *out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *SecretReference) DeepCopyInto(out *SecretReference) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SecretReference.
func (in *SecretReference) DeepCopy() *SecretReference {
	if in == nil {
		return nil
	}
	out := new(SecretReference)
	in.DeepCopyInto(out)
	return out
}

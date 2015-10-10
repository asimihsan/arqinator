/*
arqinator: arq/types/connector.go
Implements an Arq Connector, an interface for backup types (e.g. S3 and Google Cloud Storage).

Copyright 2015 Asim Ihsan

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

package connector

type Object interface {
	GetPath() string
}

type Connection interface {
	String() string
	GetCacheDirectory() string
	ListObjectsAsFolders(prefix string) ([]Object, error)
	ListObjectsAsAll(prefix string) ([]Object, error)
	Get(key string) (string, error)
	CachedGet(key string) (string, error)
}

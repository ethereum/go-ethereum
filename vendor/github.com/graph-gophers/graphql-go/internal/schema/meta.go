package schema

func init() {
	_ = newMeta()
}

// newMeta initializes an instance of the meta Schema.
func newMeta() *Schema {
	s := &Schema{
		entryPointNames: make(map[string]string),
		Types:           make(map[string]NamedType),
		Directives:      make(map[string]*DirectiveDecl),
	}
	if err := s.Parse(metaSrc, false); err != nil {
		panic(err)
	}
	return s
}

var metaSrc = `
	# The ` + "`" + `Int` + "`" + ` scalar type represents non-fractional signed whole numeric values. Int can represent values between -(2^31) and 2^31 - 1.
	scalar Int

	# The ` + "`" + `Float` + "`" + ` scalar type represents signed double-precision fractional values as specified by [IEEE 754](http://en.wikipedia.org/wiki/IEEE_floating_point).
	scalar Float

	# The ` + "`" + `String` + "`" + ` scalar type represents textual data, represented as UTF-8 character sequences. The String type is most often used by GraphQL to represent free-form human-readable text.
	scalar String

	# The ` + "`" + `Boolean` + "`" + ` scalar type represents ` + "`" + `true` + "`" + ` or ` + "`" + `false` + "`" + `.
	scalar Boolean

	# The ` + "`" + `ID` + "`" + ` scalar type represents a unique identifier, often used to refetch an object or as key for a cache. The ID type appears in a JSON response as a String; however, it is not intended to be human-readable. When expected as an input type, any string (such as ` + "`" + `"4"` + "`" + `) or integer (such as ` + "`" + `4` + "`" + `) input value will be accepted as an ID.
	scalar ID

	# Directs the executor to include this field or fragment only when the ` + "`" + `if` + "`" + ` argument is true.
	directive @include(
		# Included when true.
		if: Boolean!
	) on FIELD | FRAGMENT_SPREAD | INLINE_FRAGMENT

	# Directs the executor to skip this field or fragment when the ` + "`" + `if` + "`" + ` argument is true.
	directive @skip(
		# Skipped when true.
		if: Boolean!
	) on FIELD | FRAGMENT_SPREAD | INLINE_FRAGMENT

	# Marks an element of a GraphQL schema as no longer supported.
	directive @deprecated(
		# Explains why this element was deprecated, usually also including a suggestion
		# for how to access supported similar data. Formatted in
		# [Markdown](https://daringfireball.net/projects/markdown/).
		reason: String = "No longer supported"
	) on FIELD_DEFINITION | ENUM_VALUE

	# A Directive provides a way to describe alternate runtime execution and type validation behavior in a GraphQL document.
	#
	# In some cases, you need to provide options to alter GraphQL's execution behavior
	# in ways field arguments will not suffice, such as conditionally including or
	# skipping a field. Directives provide this by describing additional information
	# to the executor.
	type __Directive {
		name: String!
		description: String
		locations: [__DirectiveLocation!]!
		args: [__InputValue!]!
	}

	# A Directive can be adjacent to many parts of the GraphQL language, a
	# __DirectiveLocation describes one such possible adjacencies.
	enum __DirectiveLocation {
		# Location adjacent to a query operation.
		QUERY
		# Location adjacent to a mutation operation.
		MUTATION
		# Location adjacent to a subscription operation.
		SUBSCRIPTION
		# Location adjacent to a field.
		FIELD
		# Location adjacent to a fragment definition.
		FRAGMENT_DEFINITION
		# Location adjacent to a fragment spread.
		FRAGMENT_SPREAD
		# Location adjacent to an inline fragment.
		INLINE_FRAGMENT
		# Location adjacent to a schema definition.
		SCHEMA
		# Location adjacent to a scalar definition.
		SCALAR
		# Location adjacent to an object type definition.
		OBJECT
		# Location adjacent to a field definition.
		FIELD_DEFINITION
		# Location adjacent to an argument definition.
		ARGUMENT_DEFINITION
		# Location adjacent to an interface definition.
		INTERFACE
		# Location adjacent to a union definition.
		UNION
		# Location adjacent to an enum definition.
		ENUM
		# Location adjacent to an enum value definition.
		ENUM_VALUE
		# Location adjacent to an input object type definition.
		INPUT_OBJECT
		# Location adjacent to an input object field definition.
		INPUT_FIELD_DEFINITION
	}

	# One possible value for a given Enum. Enum values are unique values, not a
	# placeholder for a string or numeric value. However an Enum value is returned in
	# a JSON response as a string.
	type __EnumValue {
		name: String!
		description: String
		isDeprecated: Boolean!
		deprecationReason: String
	}

	# Object and Interface types are described by a list of Fields, each of which has
	# a name, potentially a list of arguments, and a return type.
	type __Field {
		name: String!
		description: String
		args: [__InputValue!]!
		type: __Type!
		isDeprecated: Boolean!
		deprecationReason: String
	}

	# Arguments provided to Fields or Directives and the input fields of an
	# InputObject are represented as Input Values which describe their type and
	# optionally a default value.
	type __InputValue {
		name: String!
		description: String
		type: __Type!
		# A GraphQL-formatted string representing the default value for this input value.
		defaultValue: String
	}

	# A GraphQL Schema defines the capabilities of a GraphQL server. It exposes all
	# available types and directives on the server, as well as the entry points for
	# query, mutation, and subscription operations.
	type __Schema {
		# A list of all types supported by this server.
		types: [__Type!]!
		# The type that query operations will be rooted at.
		queryType: __Type!
		# If this server supports mutation, the type that mutation operations will be rooted at.
		mutationType: __Type
		# If this server support subscription, the type that subscription operations will be rooted at.
		subscriptionType: __Type
		# A list of all directives supported by this server.
		directives: [__Directive!]!
	}

	# The fundamental unit of any GraphQL Schema is the type. There are many kinds of
	# types in GraphQL as represented by the ` + "`" + `__TypeKind` + "`" + ` enum.
	#
	# Depending on the kind of a type, certain fields describe information about that
	# type. Scalar types provide no information beyond a name and description, while
	# Enum types provide their values. Object and Interface types provide the fields
	# they describe. Abstract types, Union and Interface, provide the Object types
	# possible at runtime. List and NonNull types compose other types.
	type __Type {
		kind: __TypeKind!
		name: String
		description: String
		fields(includeDeprecated: Boolean = false): [__Field!]
		interfaces: [__Type!]
		possibleTypes: [__Type!]
		enumValues(includeDeprecated: Boolean = false): [__EnumValue!]
		inputFields: [__InputValue!]
		ofType: __Type
	}

	# An enum describing what kind of type a given ` + "`" + `__Type` + "`" + ` is.
	enum __TypeKind {
		# Indicates this type is a scalar.
		SCALAR
		# Indicates this type is an object. ` + "`" + `fields` + "`" + ` and ` + "`" + `interfaces` + "`" + ` are valid fields.
		OBJECT
		# Indicates this type is an interface. ` + "`" + `fields` + "`" + ` and ` + "`" + `possibleTypes` + "`" + ` are valid fields.
		INTERFACE
		# Indicates this type is a union. ` + "`" + `possibleTypes` + "`" + ` is a valid field.
		UNION
		# Indicates this type is an enum. ` + "`" + `enumValues` + "`" + ` is a valid field.
		ENUM
		# Indicates this type is an input object. ` + "`" + `inputFields` + "`" + ` is a valid field.
		INPUT_OBJECT
		# Indicates this type is a list. ` + "`" + `ofType` + "`" + ` is a valid field.
		LIST
		# Indicates this type is a non-null. ` + "`" + `ofType` + "`" + ` is a valid field.
		NON_NULL
	}
`

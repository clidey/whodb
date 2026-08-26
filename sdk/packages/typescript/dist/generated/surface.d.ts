/** Facade surface mapping generated from sdk/spec/surface.yaml. */
export declare const surface: {
    readonly ontology: {
        readonly entities: {
            readonly operation: "OntologyEntities";
            readonly autofill: {
                readonly projectId: "$project";
            };
            readonly rename: {};
            readonly paginated: null;
        };
        readonly entity: {
            readonly operation: "OntologyEntity";
            readonly autofill: {
                readonly projectId: "$project";
            };
            readonly rename: {};
            readonly paginated: null;
        };
        readonly describe: {
            readonly operation: "OntologyDescribe";
            readonly autofill: {
                readonly projectId: "$project";
            };
            readonly rename: {};
            readonly paginated: null;
        };
        readonly list: {
            readonly operation: "OntologyRows";
            readonly autofill: {
                readonly projectId: "$project";
            };
            readonly rename: {};
            readonly paginated: {
                readonly pageSizeArg: "pageSize";
                readonly pageOffsetArg: "pageOffset";
            };
        };
        readonly query: {
            readonly operation: "OntologyQuery";
            readonly autofill: {
                readonly projectId: "$project";
            };
            readonly rename: {};
            readonly paginated: null;
        };
        readonly aggregate: {
            readonly operation: "OntologyAggregate";
            readonly autofill: {
                readonly projectId: "$project";
            };
            readonly rename: {};
            readonly paginated: {
                readonly pageSizeArg: "pageSize";
                readonly pageOffsetArg: "pageOffset";
            };
        };
        readonly stats: {
            readonly operation: "OntologyStats";
            readonly autofill: {
                readonly projectId: "$project";
            };
            readonly rename: {};
            readonly paginated: null;
        };
        readonly similar: {
            readonly operation: "OntologySimilar";
            readonly autofill: {
                readonly projectId: "$project";
            };
            readonly rename: {};
            readonly paginated: null;
        };
        readonly followLink: {
            readonly operation: "OntologyFollowLink";
            readonly autofill: {
                readonly projectId: "$project";
            };
            readonly rename: {};
            readonly paginated: {
                readonly pageSizeArg: "pageSize";
                readonly pageOffsetArg: "pageOffset";
            };
        };
        readonly followIncomingLink: {
            readonly operation: "OntologyFollowIncomingLink";
            readonly autofill: {
                readonly projectId: "$project";
            };
            readonly rename: {};
            readonly paginated: {
                readonly pageSizeArg: "pageSize";
                readonly pageOffsetArg: "pageOffset";
            };
        };
        readonly fastLookups: {
            readonly operation: "OntologyFastLookups";
            readonly autofill: {
                readonly projectId: "$project";
            };
            readonly rename: {};
            readonly paginated: null;
        };
        readonly create: {
            readonly operation: "OntologyAddRow";
            readonly autofill: {
                readonly projectId: "$project";
            };
            readonly rename: {};
            readonly paginated: null;
        };
        readonly createMany: {
            readonly operation: "OntologyAddRows";
            readonly autofill: {
                readonly projectId: "$project";
            };
            readonly rename: {};
            readonly paginated: null;
        };
        readonly update: {
            readonly operation: "OntologyUpdateRow";
            readonly autofill: {
                readonly projectId: "$project";
            };
            readonly rename: {};
            readonly paginated: null;
        };
        readonly delete: {
            readonly operation: "OntologyDeleteRow";
            readonly autofill: {
                readonly projectId: "$project";
            };
            readonly rename: {};
            readonly paginated: null;
        };
    };
    readonly dataset: {
        readonly list: {
            readonly operation: "ProjectDatasets";
            readonly autofill: {
                readonly projectId: "$project";
            };
            readonly rename: {};
            readonly paginated: null;
        };
        readonly rows: {
            readonly operation: "QueryDataset";
            readonly autofill: {};
            readonly rename: {};
            readonly paginated: null;
        };
    };
    readonly source: {
        readonly list: {
            readonly operation: "ProjectSources";
            readonly autofill: {
                readonly projectId: "$project";
            };
            readonly rename: {};
            readonly paginated: null;
        };
        readonly rows: {
            readonly operation: "PlatformSourceRows";
            readonly autofill: {
                readonly projectId: "$project";
            };
            readonly rename: {};
            readonly paginated: {
                readonly pageSizeArg: "pageSize";
                readonly pageOffsetArg: "pageOffset";
            };
        };
        readonly objects: {
            readonly operation: "PlatformSourceObjects";
            readonly autofill: {
                readonly projectId: "$project";
            };
            readonly rename: {};
            readonly paginated: null;
        };
        readonly columns: {
            readonly operation: "PlatformSourceColumns";
            readonly autofill: {
                readonly projectId: "$project";
            };
            readonly rename: {};
            readonly paginated: null;
        };
    };
};

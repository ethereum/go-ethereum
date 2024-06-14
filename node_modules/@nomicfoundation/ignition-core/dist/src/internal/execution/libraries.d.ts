/**
 * This file has functions to handle libraries validation and linking.
 *
 * The functions in this file follow the same format that Hardhat uses
 * to name libraries. That is, they receive a map from library names to
 * addresses, where the name can one of:
 *  * The name of the library, if it's unambiguous.
 *  * The fully qualified name of the library, if it's ambiguous.
 *
 * The functions throw in the case of ambiguity, indicating the user
 * how to fix it.
 *
 * @file
 */
import { IgnitionError } from "../../errors";
import { Artifact } from "../../types/artifact";
/**
 * This function validates that the libraries object ensures that libraries:
 *  - Are not repeated (i.e. only the FQN or bare name should be used).
 *  - Are needed by the contract.
 *  - Are not ambiguous.
 *  - Are not missing.
 */
export declare function validateLibraryNames(artifact: Artifact, libraryNames: string[]): IgnitionError[];
/**
 * Links the libaries in the artifact's deployment bytecode, trowing if the
 * libraries object is invalid.
 */
export declare function linkLibraries(artifact: Artifact, libraries: {
    [libraryName: string]: string;
}): string;
//# sourceMappingURL=libraries.d.ts.map
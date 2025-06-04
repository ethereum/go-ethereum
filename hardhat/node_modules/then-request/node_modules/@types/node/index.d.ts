// Type definitions for Node.js 8.10
// Project: http://nodejs.org/
// Definitions by: Microsoft TypeScript <https://github.com/Microsoft>
//                 DefinitelyTyped <https://github.com/DefinitelyTyped>
//                 Parambir Singh <https://github.com/parambirs>
//                 Wilco Bakker <https://github.com/WilcoBakker>
//                 Chigozirim C. <https://github.com/smac89>
//                 Flarna <https://github.com/Flarna>
//                 Mariusz Wiktorczyk <https://github.com/mwiktorczyk>
//                 wwwy3y3 <https://github.com/wwwy3y3>
//                 Deividas Bakanas <https://github.com/DeividasBakanas>
//                 Kelvin Jin <https://github.com/kjin>
//                 Alvis HT Tang <https://github.com/alvis>
//                 Sebastian Silbermann <https://github.com/eps1lon>
//                 Hannes Magnusson <https://github.com/Hannes-Magnusson-CK>
//                 Alberto Schiabel <https://github.com/jkomyno>
//                 Huw <https://github.com/hoo29>
//                 Nicolas Even <https://github.com/n-e>
//                 Nikita Galkin <https://github.com/galkin>
//                 Bruno Scheufler <https://github.com/brunoscheufler>
//                 Hoàng Văn Khải <https://github.com/KSXGitHub>
//                 Lishude <https://github.com/islishude>
//                 Andrew Makarov <https://github.com/r3nya>
//                 Jordi Oliveras Rovira <https://github.com/j-oliveras>
//                 Thanik Bhongbhibhat <https://github.com/bhongy>
// Definitions: https://github.com/DefinitelyTyped/DefinitelyTyped
// NOTE: These definitions support NodeJS and TypeScript 3.2.

// NOTE: TypeScript version-specific augmentations can be found in the following paths:
//          - ~/base.d.ts         - Shared definitions common to all TypeScript versions
//          - ~/index.d.ts        - Definitions specific to TypeScript 2.1
//          - ~/ts3.2/index.d.ts  - Definitions specific to TypeScript 3.2

// Reference required types from the default lib:
/// <reference lib="es2017" />

// Base definitions for all NodeJS modules that are not specific to any version of TypeScript:
/// <reference path="base.d.ts" />

// TypeScript 3.2-specific augmentations:
declare module "util" {
    namespace inspect {
        const custom: unique symbol;
    }
    namespace promisify {
        const custom: unique symbol;
    }
}

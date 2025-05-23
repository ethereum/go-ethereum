import { use } from "chai";
import chaiAsPromised from "chai-as-promised";

import "../types";
import { hardhatChaiMatchers } from "./hardhatChaiMatchers";

use(hardhatChaiMatchers);
use(chaiAsPromised);

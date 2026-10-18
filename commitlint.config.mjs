import { realpathSync } from 'node:fs';
import { createRequire } from 'node:module';

// Resolve the preset beside the CLI installed by mise.
const require = createRequire(realpathSync(process.argv[1]));

export default {
  extends: [require.resolve('@commitlint/config-conventional')],
};

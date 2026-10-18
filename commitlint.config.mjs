import { realpathSync } from 'node:fs';
import { createRequire } from 'node:module';

// The preset lives with the commitlint CLI installed by mise, not in the checkout.
const require = createRequire(realpathSync(process.argv[1]));

export default {
  extends: [require.resolve('@commitlint/config-conventional')],
  // PR titles are never exempted as generated merge, revert, or fixup messages.
  defaultIgnores: false,
  rules: {
    // Conventional Commits 1.0.0 has no line-length or scope-case restriction.
    'header-max-length': [0],
    'body-max-line-length': [0],
    'footer-max-line-length': [0],
    'scope-case': [0],

    // The five additional rules selected for this repository.
    'type-enum': [
      2, 'always',
      ['build', 'chore', 'ci', 'docs', 'feat', 'fix', 'perf', 'refactor', 'revert', 'style', 'test'],
    ],
    'type-case': [2, 'always', 'lower-case'],
    'subject-case': [2, 'never', ['sentence-case', 'start-case', 'pascal-case', 'upper-case']],
    'subject-full-stop': [2, 'never', '.'],
    'header-trim': [2, 'always'],
  },
};

import commitTemplate from "./.github/release-commit.template.mjs";

export default {
  branches: [
    {
      name: "release/v+([0-9])?(.{+([0-9]),x}).x",
      range: "${name.replace(/^release\\/v/g, '')}",
      channel: "${name.replace(/release\\/(v[0-9]+)\\..*/, '$1-lts')}",
    },
    "main",
    "next",
  ],
  plugins: [
    "@semantic-release/commit-analyzer",
    [
      "@semantic-release/release-notes-generator",
      {
        writerOpts: {
          commitPartial: commitTemplate,
        },
      },
    ],
    [
      "@semantic-release/exec",
      {
        prepareCmd: "./scripts/build.sh ${nextRelease.version}",
      },
    ],
    [
      "@semantic-release/github",
      {
        assets: [
          { path: "dist/*.tar.gz" },
          { path: "dist/*.zip" },
          { path: "dist/checksums.txt" },
        ],
      },
    ],
  ],
};

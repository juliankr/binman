module.exports = {
  platform: 'github',
  token: process.env.GITHUB_TOKEN,
  repositories: [
    'juliankr/binman'
  ],
  regexManagers: [
    {
      fileMatch: ["binman.yaml"],
      matchStrings: [
        "# renovate: datasource=(?<datasource>\\S+) depName=(?<depName>\\S+)[\\s\\S]*?version: (?<currentValue>.*)"
      ],
      datasourceTemplate: "{{{datasource}}}",
      depNameTemplate: "{{{depName}}}",
    }
  ]
};
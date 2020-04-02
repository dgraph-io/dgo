## Steps to Release a Minor Version

1. Checkout the commit that you want to tag with the new release.

2. Run the following command to create an annotated tag:

```
git tag -a <new tag>
```

3. Push the tag to GitHub:

```
git push origin <new tag>
```

## Steps to Release a Major Version

1. Update the `go.mod` file to the module name with the correct version.

2. Change all the import paths to import `v<new major version>`.

For example, current import path is `"github.com/dgraph-io/dgo/v200"`.
When we release v201.07.0, we would replace the import paths to `"github.com/dgraph-io/dgo/v201"`.

3. Update [Supported Version](https://github.com/dgraph-io/dgo/#supported-versions).

4. Commit all the changes and get them merged to master branch.

Now, follow the [steps to release Minor Version](#steps-to-release-a-minor-version) as above.

Note that, now you have to also change the import path in the applications that use `dgo` including `dgraph`.
If you are doing the release, it would be nice that you raise a PR on Dgraph repo as well updating the paths.

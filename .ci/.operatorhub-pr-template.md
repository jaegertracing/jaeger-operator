Thanks submitting your Operator. Please check below list before you create your Pull Request.
*************************************************
**Flat operator directory structure is obsolete from 23-rd of October 2019, only nested directory structure will be accepted.**
*************************************************

### New Submissions

* [ ] Has you operator [nested directory structure](https://github.com/operator-framework/community-operators/blob/master/docs/contributing.md#create-a-bundle)?
* [ ] Have you selected the Project *Community Operator Submissions* in your PR on the right-hand menu bar?
* [ ] Are you familiar with our [contribution guidelines](https://github.com/operator-framework/community-operators/blob/master/docs/contributing.md)?
* [ ] Have you [packaged and deployed](https://github.com/operator-framework/community-operators/blob/master/docs/testing-operators.md) your Operator for Operator Framework?
* [ ] Have you tested your Operator with all Custom Resource Definitions?
* [ ] Have you tested your Operator in all supported [installation modes](https://github.com/operator-framework/operator-lifecycle-manager/blob/master/doc/design/building-your-csv.md#operator-metadata)?
* [ ] Is your submission [signed](https://github.com/operator-framework/community-operators/blob/master/docs/contributing.md#sign-your-work)?

### Updates to existing Operators

* [ ] Is your new CSV pointing to the previous version with the `replaces` property?
* [ ] Is your new CSV referenced in the [appropriate channel](https://github.com/operator-framework/community-operators/blob/master/docs/contributing.md#bundle-format) defined in the `package.yaml` ?
* [ ] Have you tested an update to your Operator when deployed via OLM?
* [ ] Is your submission [signed](https://github.com/operator-framework/community-operators/blob/master/docs/contributing.md#sign-your-work)?

### Your submission should not

* [ ] Modify more than one operator
* [ ] Modify an Operator you don't own
* [ ] Rename an operator - please remove and add with a different name instead
* [ ] Submit operators to both `upstream-community-operators` and `community-operators` at once
* [ ] Modify any files outside the above mentioned folders
* [ ] Contain more than one commit. **Please squash your commits.**

### Operator Description must contain (in order)

1. [ ] Description about the managed Application and where to find more information
2. [ ] Features and capabilities of your Operator and how to use it
3. [ ] Any manual steps about potential pre-requisites for using your Operator

### Operator Metadata should contain

* [ ] Human readable name and 1-liner description about your Operator
* [ ] Valid [category name](https://github.com/operator-framework/community-operators/blob/master/docs/required-fields.md#categories)<sup>1</sup>
* [ ] One of the pre-defined [capability levels](https://github.com/operator-framework/operator-courier/blob/4d1a25d2c8d52f7de6297ec18d8afd6521236aa2/operatorcourier/validate.py#L556)<sup>2</sup>
* [ ] Links to the maintainer, source code and documentation
* [ ] Example templates for all Custom Resource Definitions intended to be used
* [ ] A quadratic logo

Remember that you can preview your CSV [here](https://operatorhub.io/preview).

--

<sup>1</sup> If you feel your Operator does not fit any of the pre-defined categories, file a PR against this repo and explain your need

<sup>2</sup> For more information see [here](https://github.com/operator-framework/operator-sdk/blob/master/doc/images/operator-capability-level.svg)

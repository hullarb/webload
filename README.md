# webload

webload is a simple command line tool to upload your static webcontent (or other files) to an aws S3 compatible storage. It can help if you need to upload/synchronize a lot of files.

webload can synchronize a local directory with an S3 bucket and it can compress the uploaded files on the fly, set the content type and content encoding automaically.

With compressing the web contnet one can save on storage space and also web load times.

Usage: `AWS_ACCESS_KEY=your_aws_access_key AWS_SECRET_KEY=your_aws_secret_key  go run . -b your_bucket_name -r your_bucket_region  [-c gzip] -d your_local_directory_to_sync [-s]`

Optional parameters: 
* `-c` enables compression, currently only `gzip` value is accepted, this will set the content-encoding of the uploaded files to gzip.
* `-s` for synchronizing, which means after uploading the content from the local folder all the localny not existing content will be deleted from the bucket.

For details how to set up a aws S3 bucket for web hosting you can check the [official documentation](https://docs.aws.amazon.com/AmazonS3/latest/dev/WebsiteHosting.html).

## Note: all uploaded object will have public read acl.
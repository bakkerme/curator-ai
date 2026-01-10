# PostBlock

A PostBlock contains the data and metadata of a Post, including everything needed to represent and operate on the Post

PostBlock
* ID (string)
* URL (string)
* Comment (CommentBlock)
* WebBlocks
* ImageBlocks
* Summary
* Quality

# CommentBlock

Contains data and metadata representing a Comment attached to a Post

CommentBlock
* Comments ([]string)
* URLs (WebBlock)
* Images (ImageBlock)
* WasSummarised (bool)
* Summary (string)
* Quality

# WebBlock

Contains the data and metadata of a website. This can represent a URL attached to a post and can optionally be used to scrape data from that page for further processing

WebBlock
* URL (string)
* WasFetched (bool)
* Page (string)
* Request 
* Summary (string)
* WasSummarised (bool)
* Quality

# ImageBlock

Contains the data and metadata of an Image, including a URL source. An Image might start life as a URL parsed from another Block that matches existing patterns for a URL that contains an imeage.

ImageBlock
* URL (string)
* ImageData
* WasFetched (bool)
* Summary (string)
* WasSummarised (bool)
* Quality
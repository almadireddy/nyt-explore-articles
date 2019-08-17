package main

import (
	"cloud.google.com/go/firestore"
	"context"
	"google.golang.org/api/iterator"
)

type Store interface {
	GetArticles(ctx context.Context) ([]*Article, error)
	GetArticlesByLocation(ctx context.Context, location string) ([]*Article, error)
	GetImages(ctx context.Context) ([]*Image, error)
	SaveArticles(ctx context.Context, articles []*Article) error
	SaveArticle(ctx context.Context, article *Article) error
}

type FirestoreDb struct {
	client *firestore.Client
}

func (f *FirestoreDb) GetArticles(ctx context.Context) ([]*Article, error) {
	var articles []*Article
	iter := f.client.Collection("articles").Documents(ctx)

	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		var article Article
		err = doc.DataTo(&article)
		if err != nil {
			return nil, err
		}
		articles = append(articles, &article)
	}

	return articles, nil
}

func (f *FirestoreDb) GetArticlesByLocation(ctx context.Context, location string) ([]*Article, error) {
	var articles []*Article

	iter := f.client.Collection("articles").Where("generalLocation", "==", location).OrderBy("createdAt", firestore.Desc).Documents(ctx)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		var article Article
		err = doc.DataTo(&article)
		if err != nil {
			return nil, err
		}
		articles = append(articles, &article)
	}

	return articles, nil
}


func (f *FirestoreDb) GetImages(ctx context.Context) ([]*Image, error) {
	var images []*Image

	iter := f.client.Collection("images").Documents(ctx)

	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		var image Image
		err = doc.DataTo(&image)
		if err != nil {
			return nil, err
		}
		images = append(images, &image)
	}

	return images, nil
}
  
func (f *FirestoreDb) SaveArticles(ctx context.Context, articles []*Article) error {
	for _, a := range articles {
		err := f.SaveArticle(ctx, a)
		if err != nil {
			return err
		}
	}

	return nil
}

func (f *FirestoreDb) SaveArticle(ctx context.Context, article *Article) error {
	_, _, err := f.client.Collection("articles").Add(ctx, article)
	return err

}

var articleStore Store

func InitArticleStore(s Store) {
	articleStore = s
}

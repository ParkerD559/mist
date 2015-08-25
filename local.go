// Copyright (c) 2015 Pagoda Box Inc
//
// This Source Code Form is subject to the terms of the Mozilla Public License, v.
// 2.0. If a copy of the MPL was not distributed with this file, You can obtain one
// at http://mozilla.org/MPL/2.0/.
//

package mist

import (
	set "github.com/deckarep/golang-set"
)

func makeSet(tags []string) set.Set {
	set := set.NewThreadUnsafeSet()
	for _, i := range tags {
		set.Add(i)
	}

	return set
}

func NewLocalClient(mist *Mist, buffer int) *subscriber {
	client := &subscriber{
		check: make(chan Message, buffer),
		done:  make(chan bool),
		pipe:  make(chan Message),
		mist:  mist,
		id:    mist.nextId()}

	// this gofunc handles matching messages to subscriptions for the client
	go func(client *subscriber) {

		defer func() {
			close(client.check)
			close(client.pipe)
		}()

		for {
			select {
			case msg := <-client.check:
				// we do this so that we don't need a mutex
				subscriptions := client.subscriptions
				for _, subscription := range subscriptions {
					if subscription.IsSubset(msg.tags) {
						client.pipe <- msg
					}
				}
			case <-client.done:
				return
			}
		}
	}(client)

	mist.addSubscriber(client)
	return client
}

func (client *subscriber) Subscribe(tags []string) {
	subscription := makeSet(tags)

	client.Lock()
	client.subscriptions = append(client.subscriptions, subscription)
	client.Unlock()
}

// Unsubscribe iterates through each of mist clients subscriptions keeping all subscriptions
// that aren't the specified subscription
func (client *subscriber) Unsubscribe(tags []string) {
	client.Lock()

	//create a set for quick comparison
	test := makeSet(tags)

	// create a slice of subscriptions that are going to be kept
	keep := []set.Set{}

	// iterate over all of mist clients subscriptions looking for ones that match the
	// subscription to unsubscribe
	for _, subscription := range client.subscriptions {

		// if they are not the same set (meaning they are a different subscription) then add them
		// to the keep set
		if !test.Equal(subscription) {
			keep = append(keep, subscription)
		}
	}

	client.subscriptions = keep

	client.Unlock()
}

func (client *subscriber) List() ([][]string, error) {
	subscriptions := make([][]string, len(client.subscriptions))
	for i, subscription := range client.subscriptions {
		sub := make([]string, subscription.Cardinality())
		for j, tag := range subscription.ToSlice() {
			sub[j] = tag.(string)
		}
		subscriptions[i] = sub
	}
	return subscriptions, nil
}

func (client *subscriber) Close() error {
	// this closes the goroutine that is matching messages to subscriptions
	close(client.done)

	client.mist.removeSubscriber(client.id)
	return nil
}

// Returns all messages that have sucessfully matched the list of subscriptions that this
// client has subscribed to
func (client *subscriber) Messages() <-chan Message {
	return client.pipe
}

// Sends a message across mist
func (client *subscriber) Publish(tags []string, data interface{}) error {
	client.mist.Publish(tags, data)
	return nil
}

func (client *subscriber) Ping() error {
	return nil
}

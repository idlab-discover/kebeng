package store

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/snapcore/snapd/asserts"
)

// assertions started working on

// SnapRevision
// SnapDeclaration
// Account
// AccountKey

func (s *Store) getSnapRevisionAssertion(c *gin.Context) {
	sha3384digest := c.Param("sha3384digest")
	logrus.Tracef("Requested snap-revision: %s", sha3384digest)

	rootAuthorityUUID, err := uuid.Parse(s.rootAuthorityID)
	if err != nil {
		logrus.Errorf("Invalid root authority ID: %s", s.rootAuthorityID)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	assertions, err := s.handler.GetSnapRevisionAssertion(sha3384digest, s.rootStoreKey, s.assertsDatabase, rootAuthorityUUID)
	if err != nil {
		logrus.Errorf("Failed to get snap-revision assertion: %s", err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	if assertions == nil {
		logrus.Errorf("Snap-revision assertion not found")
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	encodedAssertion := asserts.Encode(assertions)
	logrus.Tracef("Sending snap-revision assertion: %s", string(encodedAssertion))

	c.Writer.Header().Set("Content-Type", asserts.MediaType) // MediaType is the type for encoded assertions
	c.Writer.WriteHeader(http.StatusOK)

	if _, err := c.Writer.Write(encodedAssertion); err != nil {
		logrus.Errorf("Failed to write snap-revision assertion: %s", err)
		c.AbortWithStatus(http.StatusInternalServerError)
	}
}

func (s *Store) getSnapDeclarationAssertion(c *gin.Context) {
	snapId := c.Param("snap-id")
	logrus.Tracef("Requested snap-declaration: %s", snapId)

	snapUUID, err := uuid.Parse(snapId)
	if err != nil {
		logrus.Errorf("Invalid snap-id: %s", snapId)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	assertion, err := s.handler.GetSnapDeclarationAssertion(snapUUID, s.rootStoreKey, s.assertsDatabase, s.rootAuthorityID)
	if err != nil {
		logrus.Errorf("Failed to get snap-declaration assertion: %v", err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	if assertion == nil {
		logrus.Warnf("Snap-declaration not found for snap-id: %s", snapId)
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	encodedAssertion := asserts.Encode(assertion)
	logrus.Tracef("Sending snap-declaration assertion: %s", string(encodedAssertion))

	// Set response headers and send the assertion
	c.Writer.Header().Set("Content-Type", asserts.MediaType)
	if _, err := c.Writer.Write(encodedAssertion); err != nil {
		logrus.Errorf("Failed to write response: %v", err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	c.Writer.WriteHeader(http.StatusOK)
}

func (s *Store) getAccountAssertion(c *gin.Context) {
	id := c.Param("id")
	logrus.Tracef("Requested account: %s", id)

	accountAssertion, err := s.handler.GetAccountAssertion(id, s.rootStoreKey, s.signingDB)
	if err == nil && accountAssertion != nil {
		assertionBytes := asserts.Encode(accountAssertion)
		c.Writer.Header().Set("Content-Type", asserts.MediaType)
		c.Writer.WriteHeader(200)

		_, err2 := c.Writer.Write(assertionBytes)
		if err2 != nil {
			logrus.Error(err2)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		return
	} else if err != nil {
		logrus.Error(err)
	} else {
		logrus.Errorf("Unknown error encountered trying to get account assertion for account id=%s", id)
	}

	c.AbortWithStatus(http.StatusInternalServerError)
}

func (s *Store) getAccountKeyAssertion(c *gin.Context) {
	key := c.Param("key")
	logrus.Tracef("Requested account-key: %s", key)

	accountKeyAssertion, err := s.handler.GetAccountKeyAssertion(key, s.rootStoreKey, s.signingDB)
	if err == nil && accountKeyAssertion != nil {
		logrus.Tracef("Found account-key assertion: %+v", accountKeyAssertion)

		c.Writer.WriteHeader(200)
		assertionBytes := asserts.Encode(accountKeyAssertion)
		_, err = c.Writer.Write(assertionBytes)
		if err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		return
	} else if err != nil {
		logrus.Error(err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	logrus.Error("Unknown error encountered while trying to get account key")
	c.AbortWithStatus(http.StatusInternalServerError)
}

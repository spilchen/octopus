package fetcher_test

import (
	"context"
	"testing"

	"github.com/kyma-incubator/octopus/pkg/humanerr"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"errors"

	"github.com/kyma-incubator/octopus/pkg/apis/testing/v1alpha1"
	"github.com/kyma-incubator/octopus/pkg/fetcher"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestFindMatching(t *testing.T) {
	sch, err := v1alpha1.SchemeBuilder.Build()
	require.NoError(t, err)

	t.Run("return all if no selectors specified", func(t *testing.T) {
		// GIVEN
		fakeCli := fake.NewFakeClientWithScheme(sch, &v1alpha1.TestDefinition{
			ObjectMeta: v1.ObjectMeta{
				Name:      "test-def",
				Namespace: "anynamespace",
			},
		})
		service := fetcher.NewForDefinition(fakeCli)

		// WHEN
		out, err := service.FindMatching(v1alpha1.ClusterTestSuite{})
		// THEN
		require.NoError(t, err)
		assert.Len(t, out, 1)
	})

	t.Run("return tests selected by names", func(t *testing.T) {
		// GIVEN
		testA := &v1alpha1.TestDefinition{
			TypeMeta: v1.TypeMeta{
				Kind:       "TestDefinition",
				APIVersion: "testing.kyma-project.io/v1alpha1",
			},
			ObjectMeta: v1.ObjectMeta{
				UID:       "test-uid-a",
				Name:      "test-a",
				Namespace: "test-a",
			},
		}
		testB := &v1alpha1.TestDefinition{
			TypeMeta: v1.TypeMeta{
				Kind:       "TestDefinition",
				APIVersion: "testing.kyma-project.io/v1alpha1",
			},
			ObjectMeta: v1.ObjectMeta{
				UID:       "test-uid-b",
				Name:      "test-b",
				Namespace: "test-b",
			},
		}
		testC := &v1alpha1.TestDefinition{
			TypeMeta: v1.TypeMeta{
				Kind:       "TestDefinition",
				APIVersion: "testing.kyma-project.io/v1alpha1",
			},
			ObjectMeta: v1.ObjectMeta{
				UID:       "test-uid-c",
				Name:      "test-c",
				Namespace: "test-c",
			},
		}

		fakeCli := fake.NewFakeClientWithScheme(sch,
			testA, testB, testC,
		)
		service := fetcher.NewForDefinition(fakeCli)
		// WHEN
		out, err := service.FindMatching(v1alpha1.ClusterTestSuite{
			Spec: v1alpha1.TestSuiteSpec{
				Selectors: v1alpha1.TestsSelector{
					MatchNames: []v1alpha1.TestDefReference{
						{
							Name:      "test-a",
							Namespace: "test-a",
						},
						{
							Name:      "test-b",
							Namespace: "test-b",
						},
					},
				},
			},
		})
		// THEN
		require.NoError(t, err)
		assert.Len(t, out, 2)
		assert.Contains(t, out, *testA)
		assert.Contains(t, out, *testB)

	})

	t.Run("return tests selected by label expressions", func(t *testing.T) {
		// GIVEN
		testA := &v1alpha1.TestDefinition{
			ObjectMeta: v1.ObjectMeta{
				UID:       "test-uid-a",
				Name:      "test-a",
				Namespace: "test-a",
				Labels: map[string]string{
					"test": "true",
				},
			},
		}
		testB := &v1alpha1.TestDefinition{
			ObjectMeta: v1.ObjectMeta{
				UID:       "test-uid-b",
				Name:      "test-b",
				Namespace: "test-b",
				Labels: map[string]string{
					"test": "false",
				},
			},
		}
		testC := &v1alpha1.TestDefinition{
			ObjectMeta: v1.ObjectMeta{
				UID:       "test-uid-c",
				Name:      "test-c",
				Namespace: "test-c",
				Labels: map[string]string{
					"other": "123",
				},
			},
		}

		fakeCli := fake.NewFakeClientWithScheme(sch,
			testA, testB, testC,
		)
		service := fetcher.NewForDefinition(fakeCli)
		// WHEN
		out, err := service.FindMatching(v1alpha1.ClusterTestSuite{
			Spec: v1alpha1.TestSuiteSpec{
				Selectors: v1alpha1.TestsSelector{
					MatchLabelExpressions: []string{
						"other",
						"test=true",
					},
				},
			},
		})
		// THEN
		require.NoError(t, err)
		assert.Len(t, out, 2)
		assert.Contains(t, out, *testA)
		assert.Contains(t, out, *testC)
	})

	t.Run("return tests returns unique result across all selectors", func(t *testing.T) {
		// GIVEN
		testA := &v1alpha1.TestDefinition{
			TypeMeta: v1.TypeMeta{
				Kind:       "TestDefinition",
				APIVersion: "testing.kyma-project.io/v1alpha1",
			},
			ObjectMeta: v1.ObjectMeta{
				UID:       "test-uid-a",
				Name:      "test-a",
				Namespace: "test-a",
				Labels: map[string]string{
					"test": "true",
				},
			},
		}

		fakeCli := fake.NewFakeClientWithScheme(sch,
			testA,
		)
		service := fetcher.NewForDefinition(fakeCli)
		// WHEN
		out, err := service.FindMatching(v1alpha1.ClusterTestSuite{
			Spec: v1alpha1.TestSuiteSpec{
				Selectors: v1alpha1.TestsSelector{
					MatchNames: []v1alpha1.TestDefReference{
						{
							Name:      "test-a",
							Namespace: "test-a",
						},
					},
					MatchLabelExpressions: []string{
						"test=true",
					},
				},
			},
		})
		// THEN
		require.NoError(t, err)
		assert.Len(t, out, 1)
		assert.Contains(t, out, *testA)
	})

	t.Run("return error if test selected by name does not exist", func(t *testing.T) {
		// GIVEN
		fakeCli := fake.NewFakeClientWithScheme(sch)
		service := fetcher.NewForDefinition(fakeCli)
		// WHEN
		_, err := service.FindMatching(v1alpha1.ClusterTestSuite{
			Spec: v1alpha1.TestSuiteSpec{
				Selectors: v1alpha1.TestsSelector{
					MatchNames: []v1alpha1.TestDefReference{
						{
							Name:      "name",
							Namespace: "ns",
						},
					},
				},
			},
		})
		// THEN
		require.EqualError(t, err, "while fetching test definition from selector [name: name, namespace: ns]: testdefinitions.testing.kyma-project.io \"name\" not found")
		herr, ok := humanerr.GetHumanReadableError(err)
		require.True(t, ok)
		assert.Equal(t, "Test Definition [name: name, namespace: ns] does not exist", herr.Message)
	})

	t.Run("return internal error when fetching selected tests failed", func(t *testing.T) {
		// GIVEN
		errClient := &mockErrReader{err: errors.New("some error")}
		service := fetcher.NewForDefinition(errClient)

		// WHEN
		_, err := service.FindMatching(v1alpha1.ClusterTestSuite{
			Spec: v1alpha1.TestSuiteSpec{
				Selectors: v1alpha1.TestsSelector{
					MatchNames: []v1alpha1.TestDefReference{
						{
							Name:      "name",
							Namespace: "ns",
						},
					},
				},
			},
		})
		// THEN
		require.EqualError(t, err, "while fetching test definition from selector [name: name, namespace: ns]: some error")
		herr, ok := humanerr.GetHumanReadableError(err)
		require.True(t, ok)
		assert.Equal(t, "Internal error", herr.Message)

	})

	t.Run("return tests in test suite order when selecting by name", func(t *testing.T) {
		// GIVEN
		testA := &v1alpha1.TestDefinition{
			TypeMeta: v1.TypeMeta{
				Kind:       "TestDefinition",
				APIVersion: "testing.kyma-project.io/v1alpha1",
			},
			ObjectMeta: v1.ObjectMeta{
				UID:       "test-uid-a",
				Name:      "test-a",
				Namespace: "ns",
				Labels: map[string]string{
					"test": "true",
				},
			},
		}

		testB := &v1alpha1.TestDefinition{
			TypeMeta: v1.TypeMeta{
				Kind:       "TestDefinition",
				APIVersion: "testing.kyma-project.io/v1alpha1",
			},
			ObjectMeta: v1.ObjectMeta{
				UID:       "test-uid-b",
				Name:      "test-b",
				Namespace: "ns",
				Labels: map[string]string{
					"test": "true",
				},
			},
		}

		testC := &v1alpha1.TestDefinition{
			TypeMeta: v1.TypeMeta{
				Kind:       "TestDefinition",
				APIVersion: "testing.kyma-project.io/v1alpha1",
			},
			ObjectMeta: v1.ObjectMeta{
				UID:       "test-uid-c",
				Name:      "test-c",
				Namespace: "ns",
				Labels: map[string]string{
					"test": "true",
				},
			},
		}

		fakeCli := fake.NewFakeClientWithScheme(sch,
			testA, testB, testC,
		)
		service := fetcher.NewForDefinition(fakeCli)
		// WHEN
		out, err := service.FindMatching(v1alpha1.ClusterTestSuite{
			Spec: v1alpha1.TestSuiteSpec{
				Selectors: v1alpha1.TestsSelector{
					MatchNames: []v1alpha1.TestDefReference{
						{
							Name:      "test-c",
							Namespace: "ns",
						},
						{
							Name:      "test-b",
							Namespace: "ns",
						},
						{
							Name:      "test-a",
							Namespace: "ns",
						},
					},
				},
			},
		})
		// THEN
		require.NoError(t, err)
		assert.Len(t, out, 3)
		expectedOrder := []v1alpha1.TestDefinition{*testC, *testB, *testA}
		assert.ElementsMatch(t, out, expectedOrder)
	})

	t.Run("return tests in test suite order when selecting by name and selector", func(t *testing.T) {
		// GIVEN
		testA := &v1alpha1.TestDefinition{
			TypeMeta: v1.TypeMeta{
				Kind:       "TestDefinition",
				APIVersion: "testing.kyma-project.io/v1alpha1",
			},
			ObjectMeta: v1.ObjectMeta{
				UID:       "test-uid-a",
				Name:      "test-a",
				Namespace: "ns",
				Labels: map[string]string{
					"test": "true",
				},
			},
		}

		testB := &v1alpha1.TestDefinition{
			TypeMeta: v1.TypeMeta{
				Kind:       "TestDefinition",
				APIVersion: "testing.kyma-project.io/v1alpha1",
			},
			ObjectMeta: v1.ObjectMeta{
				UID:       "test-uid-b",
				Name:      "test-b",
				Namespace: "ns",
				Labels: map[string]string{
					"test": "true",
				},
			},
		}

		testC := &v1alpha1.TestDefinition{
			TypeMeta: v1.TypeMeta{
				Kind:       "TestDefinition",
				APIVersion: "testing.kyma-project.io/v1alpha1",
			},
			ObjectMeta: v1.ObjectMeta{
				UID:       "test-uid-c",
				Name:      "test-c",
				Namespace: "ns",
				Labels: map[string]string{
					"test": "true",
				},
			},
		}

		fakeCli := fake.NewFakeClientWithScheme(sch,
			testA, testB, testC,
		)
		service := fetcher.NewForDefinition(fakeCli)
		// WHEN
		out, err := service.FindMatching(v1alpha1.ClusterTestSuite{
			Spec: v1alpha1.TestSuiteSpec{
				Selectors: v1alpha1.TestsSelector{
					MatchNames: []v1alpha1.TestDefReference{
						{
							Name:      "test-b",
							Namespace: "ns",
						},
					},
					MatchLabelExpressions: []string{
						"test=true",
					},
				},
			},
		})
		// THEN
		require.NoError(t, err)
		assert.Len(t, out, 3)
		expectedOrder := []v1alpha1.TestDefinition{*testB, *testA, *testC}
		assert.ElementsMatch(t, out, expectedOrder)
	})
}

type mockErrReader struct {
	err error
}

func (m *mockErrReader) Get(ctx context.Context, key client.ObjectKey, obj runtime.Object) error {
	return m.err
}

func (m *mockErrReader) List(ctx context.Context, list runtime.Object, opts ...client.ListOption) error {
	return m.err
}

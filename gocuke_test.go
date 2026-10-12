package main

import (
	"testing"

	"github.com/regen-network/gocuke"
)

func TestFeatures(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	gocuke.NewRunner(t, &assetgoblinSuite{}).
		Path("features/*.feature").
		Before((*assetgoblinSuite).Before).
		After((*assetgoblinSuite).After).
		// Server steps
		Step(`^a configured server with public directory "([^"]*)"$`, (*assetgoblinSuite).AConfiguredServerWithPublicDir).
		Step(`^a static file "([^"]*)" exists with content "([^"]*)"$`, (*assetgoblinSuite).AStaticFileExistsWithContent).
		Step(`^a configured server with image preset "([^"]*)" of width (\d+)$`, (*assetgoblinSuite).AConfiguredServerWithImagePresetOfWidth).
		Step(`^an image file "([^"]*)" exists$`, (*assetgoblinSuite).AnImageFileExists).
		Step(`^a configured server with supported image formats "([^"]*)" and "([^"]*)"$`, (*assetgoblinSuite).AConfiguredServerWithSupportedImageFormatsAnd).
		Step(`^a configured server$`, (*assetgoblinSuite).AConfiguredServer).
		Step(`^I send a GET request to "([^"]*)"$`, (*assetgoblinSuite).ISendAGetRequestTo).
		Step(`^the response status should be (\d+)$`, (*assetgoblinSuite).TheResponseStatusShouldBe).
		Step(`^the response body should contain "([^"]*)"$`, (*assetgoblinSuite).TheResponseBodyShouldContain).
		Step(`^the image response should be successful$`, (*assetgoblinSuite).TheImageResponseShouldBeSuccessful).
		// Image processing steps
		Step(`^an image service configured with supported formats "([^"]*)" and "([^"]*)"$`, (*assetgoblinSuite).AnImageServiceConfiguredWithSupportedFormatsAnd).
		Step(`^a source image "([^"]*)" exists in the image directory$`, (*assetgoblinSuite).ASourceImageExistsInTheImageDirectory).
		Step(`^an image service configured with preset "([^"]*)" of width (\d+)$`, (*assetgoblinSuite).AnImageServiceConfiguredWithPresetOfWidth).
		Step(`^I request an image with path "([^"]*)"$`, (*assetgoblinSuite).IRequestAnImageWithPath).
		Step(`^I request "([^"]*)"$`, (*assetgoblinSuite).IRequest).
		// Middleware steps
		Step(`^a signkey middleware with secret "([^"]*)"$`, (*assetgoblinSuite).ASignkeyMiddlewareWithSecret).
		Step(`^a valid token is generated for path "([^"]*)"$`, (*assetgoblinSuite).AValidTokenIsGeneratedForPath).
		Step(`^I send a GET request to "([^"]*)" with that token$`, (*assetgoblinSuite).ISendAGetRequestToWithThatToken).
		Step(`^I send a GET request to "([^"]*)" with token "([^"]*)"$`, (*assetgoblinSuite).ISendAGetRequestToWithToken).
		Step(`^I send a GET request to "([^"]*)" without a token$`, (*assetgoblinSuite).ISendAGetRequestToWithoutAToken).
		Step(`^a rate limit middleware configured for (\d+) requests per minute$`, (*assetgoblinSuite).ARateLimitMiddlewareConfiguredForRequestsPerMinute).
		Step(`^I send (\d+) requests within the time window$`, (*assetgoblinSuite).ISendRequestsWithinTheWindow).
		Step(`^I send (\d+) requests within the window$`, (*assetgoblinSuite).ISendRequestsWithinTheWindow).
		Step(`^all requests should be successful$`, (*assetgoblinSuite).AllRequestsShouldBeSuccessful).
		Step(`^a rate limit middleware configured for (\d+) requests with a (\d+)ms window$`, (*assetgoblinSuite).ARateLimitMiddlewareConfiguredForRequestsWithAMsWindow).
		Step(`^I send (\d+) requests in rapid succession$`, (*assetgoblinSuite).ISendRequestsInRapidSuccession).
		Step(`^the 4th request should be rate limited$`, (*assetgoblinSuite).The4thRequestShouldBeRateLimited).
		Step(`^I wait for the window to expire$`, (*assetgoblinSuite).IWaitForTheWindowToExpire).
		Step(`^I send another request$`, (*assetgoblinSuite).ISendAnotherRequest).
		Step(`^that request should be successful$`, (*assetgoblinSuite).ThatRequestShouldBeSuccessful).
		// Runtime plugin and codec behavior
		Step(`^the "([^"]*)" example plugin is installed$`, (*assetgoblinSuite).ExamplePluginInstalled).
		Step(`^the runtime plugins are loaded$`, (*assetgoblinSuite).LoadRuntimePlugins).
		Step(`^(\d+) plugin routes? should be registered$`, (*assetgoblinSuite).PluginRouteCount).
		Step(`^the "([^"]*)" image codec should be registered$`, (*assetgoblinSuite).ImageCodecRegistered).
		Step(`^a parent plugin serving "([^"]*)" at "([^"]*)"$`, (*assetgoblinSuite).ParentPlugin).
		Step(`^a child plugin extending the parent response with "([^"]*)"$`, (*assetgoblinSuite).ChildPlugin).
		Step(`^a plugin that extends itself$`, (*assetgoblinSuite).CyclicPlugin).
		Step(`^plugin loading should fail with "([^"]*)"$`, (*assetgoblinSuite).PluginLoadingFails).
		Step(`^the server requires the signing secret "([^"]*)"$`, (*assetgoblinSuite).ServerSigningSecret).
		Step(`^the image conversion tools are available$`, (*assetgoblinSuite).ImageConversionTools).
		Step(`^the QOA conversion tools are available$`, (*assetgoblinSuite).QOAConversionTools).
		Step(`^a PNG source "([^"]*)" of size (\d+) by (\d+) exists$`, (*assetgoblinSuite).PNGSource).
		Step(`^the response should be a QOI image of size (\d+) by (\d+)$`, (*assetgoblinSuite).QOIResponse).
		Step(`^the response should be a PNG image of size (\d+) by (\d+)$`, (*assetgoblinSuite).PNGResponse).
		Step(`^the response content type should be "([^"]*)"$`, (*assetgoblinSuite).ResponseContentType).
		Step(`^I send a HEAD request to "([^"]*)"$`, (*assetgoblinSuite).PluginHEAD).
		Step(`^I request bytes "([^"]*)" from "([^"]*)"$`, (*assetgoblinSuite).PluginRange).
		Step(`^the response should have no body and a positive content length$`, (*assetgoblinSuite).EmptyHEADResponse).
		Step(`^I save the response as image source "([^"]*)"$`, (*assetgoblinSuite).SaveImageResponse).
		Step(`^I save the response as audio source "([^"]*)"$`, (*assetgoblinSuite).SaveAudioResponse).
		Step(`^a one-second WAV audio source exists$`, (*assetgoblinSuite).WAVSource).
		Step(`^an audio source "([^"]*)" exists$`, (*assetgoblinSuite).AudioSource).
		Step(`^the response should be QOA audio with (\d+) samples$`, (*assetgoblinSuite).QOAResponse).
		Step(`^the response body should not be empty$`, (*assetgoblinSuite).NonemptyPluginResponse).
		Step(`^a one-second "([^"]*)" media source exists$`, (*assetgoblinSuite).MediaSource).
		Step(`^the response should use codec "([^"]*)" with a half-second duration$`, (*assetgoblinSuite).MediaCodecAndDuration).
		Step(`^the video response should have size (\d+) by (\d+)$`, (*assetgoblinSuite).VideoDimensions).
		Step(`^I request "([^"]*)" with a valid server signature$`, (*assetgoblinSuite).SignedPluginRequest).
		Step(`^a video source "([^"]*)" exists$`, (*assetgoblinSuite).VideoSource).
		Step(`^the response body should have length (\d+)$`, (*assetgoblinSuite).ResponseBodyLength).
		Step(`^the image route prefix is "([^"]*)"$`, (*assetgoblinSuite).ImageRoutePrefix).
		Step(`^the server is configured$`, (*assetgoblinSuite).ConfigureFeatureServer).
		Step(`^server configuration should fail with "([^"]*)"$`, (*assetgoblinSuite).ServerConfigurationFails).
		Step(`^the core image converter is available$`, (*assetgoblinSuite).CoreImageConverter).
		Step(`^the image cache should contain the transformed image$`, (*assetgoblinSuite).ImageCached).
		Step(`^I send a "([^"]*)" request to "([^"]*)"$`, (*assetgoblinSuite).RequestMethod).
		Step(`^a token generated with secret "([^"]*)" for path "([^"]*)"$`, (*assetgoblinSuite).TokenFor).
		Step(`^the server allows (\d+) requests per minute$`, (*assetgoblinSuite).ServerRateLimit).
		Step(`^the production server starts listening$`, (*assetgoblinSuite).StartProductionServer).
		Step(`^the production server is stopped$`, (*assetgoblinSuite).StopProductionServer).
		Run()
}

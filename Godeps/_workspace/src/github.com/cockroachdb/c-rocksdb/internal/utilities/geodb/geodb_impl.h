//  Copyright (c) 2013, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.
//

#ifndef ROCKSDB_LITE

#pragma once
#include <algorithm>
#include <cmath>
#include <string>
#include <sstream>
#include <stdexcept>
#include <vector>

#include "rocksdb/utilities/geo_db.h"
#include "rocksdb/utilities/stackable_db.h"
#include "rocksdb/env.h"
#include "rocksdb/status.h"

namespace rocksdb {

// A specific implementation of GeoDB

class GeoDBImpl : public GeoDB {
 public:
  GeoDBImpl(DB* db, const GeoDBOptions& options);
  ~GeoDBImpl();

  // Associate the GPS location with the identified by 'id'. The value
  // is a blob that is associated with this object.
  virtual Status Insert(const GeoObject& object) override;

  // Retrieve the value of the object located at the specified GPS
  // location and is identified by the 'id'.
  virtual Status GetByPosition(const GeoPosition& pos, const Slice& id,
                               std::string* value) override;

  // Retrieve the value of the object identified by the 'id'. This method
  // could be potentially slower than GetByPosition
  virtual Status GetById(const Slice& id, GeoObject* object) override;

  // Delete the specified object
  virtual Status Remove(const Slice& id) override;

  // Returns a list of all items within a circular radius from the
  // specified gps location
  virtual Status SearchRadial(const GeoPosition& pos, double radius,
                              std::vector<GeoObject>* values,
                              int number_of_values) override;

 private:
  DB* db_;
  const GeoDBOptions options_;
  const WriteOptions woptions_;
  const ReadOptions roptions_;

  // MSVC requires the definition for this static const to be in .CC file
  // The value of PI
  static const double PI;

  // convert degrees to radians
  static double radians(double x);

  // convert radians to degrees
  static double degrees(double x);

  // A pixel class that captures X and Y coordinates
  class Pixel {
   public:
    unsigned int x;
    unsigned int y;
    Pixel(unsigned int a, unsigned int b) :
     x(a), y(b) {
    }
  };

  // A Tile in the geoid
  class Tile {
   public:
    unsigned int x;
    unsigned int y;
    Tile(unsigned int a, unsigned int b) :
     x(a), y(b) {
    }
  };

  // convert a gps location to quad coordinate
  static std::string PositionToQuad(const GeoPosition& pos, int levelOfDetail);

  // arbitrary constant use for WGS84 via
  // http://en.wikipedia.org/wiki/World_Geodetic_System
  // http://mathforum.org/library/drmath/view/51832.html
  // http://msdn.microsoft.com/en-us/library/bb259689.aspx
  // http://www.tuicool.com/articles/NBrE73
  //
  const int Detail = 23;
  // MSVC requires the definition for this static const to be in .CC file
  static const double EarthRadius;
  static const double MinLatitude;
  static const double MaxLatitude;
  static const double MinLongitude;
  static const double MaxLongitude;

  // clips a number to the specified minimum and maximum values.
  static double clip(double n, double minValue, double maxValue) {
    return fmin(fmax(n, minValue), maxValue);
  }

  // Determines the map width and height (in pixels) at a specified level
  // of detail, from 1 (lowest detail) to 23 (highest detail).
  // Returns the map width and height in pixels.
  static unsigned int MapSize(int levelOfDetail) {
    return (unsigned int)(256 << levelOfDetail);
  }

  // Determines the ground resolution (in meters per pixel) at a specified
  // latitude and level of detail.
  // Latitude (in degrees) at which to measure the ground resolution.
  // Level of detail, from 1 (lowest detail) to 23 (highest detail).
  // Returns the ground resolution, in meters per pixel.
  static double GroundResolution(double latitude, int levelOfDetail);

  // Converts a point from latitude/longitude WGS-84 coordinates (in degrees)
  // into pixel XY coordinates at a specified level of detail.
  static Pixel PositionToPixel(const GeoPosition& pos, int levelOfDetail);

  static GeoPosition PixelToPosition(const Pixel& pixel, int levelOfDetail);

  // Converts a Pixel to a Tile
  static Tile PixelToTile(const Pixel& pixel);

  static Pixel TileToPixel(const Tile& tile);

  // Convert a Tile to a quadkey
  static std::string TileToQuadKey(const Tile& tile, int levelOfDetail);

  // Convert a quadkey to a tile and its level of detail
  static void QuadKeyToTile(std::string quadkey, Tile* tile,
                            int *levelOfDetail);

  // Return the distance between two positions on the earth
  static double distance(double lat1, double lon1,
                         double lat2, double lon2);
  static GeoPosition displaceLatLon(double lat, double lon,
                                    double deltay, double deltax);

  //
  // Returns the top left position after applying the delta to
  // the specified position
  //
  static GeoPosition boundingTopLeft(const GeoPosition& in, double radius) {
    return displaceLatLon(in.latitude, in.longitude, -radius, -radius);
  }

  //
  // Returns the bottom right position after applying the delta to
  // the specified position
  static GeoPosition boundingBottomRight(const GeoPosition& in,
                                         double radius) {
    return displaceLatLon(in.latitude, in.longitude, radius, radius);
  }

  //
  // Get all quadkeys within a radius of a specified position
  //
  Status searchQuadIds(const GeoPosition& position,
                       double radius,
                       std::vector<std::string>* quadKeys);

  //
  // Create keys for accessing rocksdb table(s)
  //
  static std::string MakeKey1(const GeoPosition& pos,
                              Slice id,
                              std::string quadkey);
  static std::string MakeKey2(Slice id);
  static std::string MakeKey1Prefix(std::string quadkey,
                                    Slice id);
  static std::string MakeQuadKeyPrefix(std::string quadkey);
};

}  // namespace rocksdb

#endif  // ROCKSDB_LITE
